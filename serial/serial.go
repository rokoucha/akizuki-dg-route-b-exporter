package serial

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/albenik/go-serial/v2"
)

var (
	ErrAlreadyStreaming = errors.New("already streaming")
	ErrCommandFailed    = errors.New("command failed")
	ErrNotStreaming     = errors.New("not streaming")
)

type Serial struct {
	listners  []*func(lines []string) error
	mu        sync.Mutex
	port      *serial.Port
	streaming atomic.Bool
}

type Config struct {
	PortName string
	BaudRate int
}

func New(c Config) (*Serial, error) {
	port, err := serial.Open(c.PortName, serial.WithBaudrate(c.BaudRate))
	if err != nil {
		return nil, err
	}

	return &Serial{
		listners: []*func(lines []string) error{},
		port:     port,
	}, nil
}

func (s *Serial) AddListner(l *func(lines []string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listners = append(s.listners, l)
}

func (s *Serial) RemoveListner(l *func(lines []string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, listner := range s.listners {
		if listner == l {
			s.listners = append(s.listners[:i], s.listners[i+1:]...)
			return
		}
	}
}

func (s *Serial) Streaming(ctx context.Context, ready chan struct{}, errCh chan error) {
	if !s.streaming.CompareAndSwap(false, true) {
		errCh <- ErrAlreadyStreaming
		return
	}
	defer s.streaming.Store(false)

	ready <- struct{}{}

	buff := make([]uint8, 255)
	data := ""
	olddata := ""
	emptyCount := 0
	for {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		default:
		}

		n, err := s.port.Read(buff)
		if err != nil {
			errCh <- err
			return
		}

		if n != 0 {
			emptyCount = 0

			data += string(buff[:n])

			continue
		}

		emptyCount++

		if emptyCount < 5 || data == olddata || strings.HasSuffix(data, "\r") {
			time.Sleep(2 * time.Duration(emptyCount) * time.Millisecond)
			continue
		}

		s.mu.Lock()
		listners := slices.Clone(s.listners)
		s.mu.Unlock()
		for _, listner := range listners {
			err := (*listner)(strings.Split(strings.TrimSuffix(data, "\r\n"), "\r\n"))
			if err != nil {
				errCh <- err
				return
			}
		}

		olddata = data
		data = ""
		emptyCount = 0
	}
}

func (s *Serial) Close() error {
	return s.port.Close()
}

func (s *Serial) Write(p []uint8) (n int, err error) {
	return s.port.Write(p)
}

func (s *Serial) Exec(ctx context.Context, command []uint8, stopper func(l []string) bool) ([]string, error) {
	if !s.streaming.Load() {
		return []string{}, ErrNotStreaming
	}

	done := make(chan struct{})
	lines := []string{}
	// リスナーは Streaming の goroutine から順に呼ばれるので、完了後は lines に触れない
	finished := false

	listener := func(l []string) error {
		if finished || len(l) == 0 {
			return nil
		}
		if stopper == nil {
			if len(l) == 1 && l[0] == "" {
				finished = true
				close(done)
				return nil
			}
		} else {
			if stopper(l) {
				lines = append(lines, l...)
				finished = true
				close(done)
				return nil
			}
		}
		lines = append(lines, l...)
		return nil
	}

	s.AddListner(&listener)

	_, err := s.port.Write(command)
	if err != nil {
		s.RemoveListner(&listener)
		return []string{}, err
	}

	select {
	case <-ctx.Done():
		s.RemoveListner(&listener)
		return []string{}, ctx.Err()
	case <-done:
		s.RemoveListner(&listener)
		return lines, nil
	}
}
