package serial

import (
	"context"
	"testing"
	"time"
)

type testPort struct{ reads chan string }

func (p *testPort) Read(b []byte) (int, error) {
	select {
	case s := <-p.reads:
		return copy(b, s), nil
	default:
		return 0, nil
	}
}
func (*testPort) Write(b []byte) (int, error) { return len(b), nil }
func (*testPort) Close() error                { return nil }

func TestStreamingRepeatedAndSplitLines(t *testing.T) {
	p := &testPort{reads: make(chan string, 1)}
	s := &Serial{port: p}
	received := make(chan []string, 4)
	listener := func(lines []string) error { received <- lines; return nil }
	s.AddListner(&listener)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	errs := make(chan error, 1)
	go s.Streaming(ctx, ready, errs)
	<-ready
	for i := 0; i < 2; i++ {
		p.reads <- "EVENT 25 FE80::1\r\n"
		select {
		case lines := <-received:
			if len(lines) != 1 || lines[0] != "EVENT 25 FE80::1" {
				t.Fatalf("unexpected lines: %q", lines)
			}
		case <-time.After(time.Second):
			t.Fatal("repeated event was not delivered")
		}
	}
	p.reads <- "EVENT 2"
	select {
	case lines := <-received:
		t.Fatalf("partial line delivered: %q", lines)
	case <-time.After(100 * time.Millisecond):
	}
	p.reads <- "9 FE80::1\r\n"
	select {
	case lines := <-received:
		if len(lines) != 1 || lines[0] != "EVENT 29 FE80::1" {
			t.Fatalf("unexpected lines: %q", lines)
		}
	case <-time.After(time.Second):
		t.Fatal("completed line was not delivered")
	}
	cancel()
	select {
	case <-errs:
	case <-time.After(time.Second):
		t.Fatal("stream did not stop")
	}
}
