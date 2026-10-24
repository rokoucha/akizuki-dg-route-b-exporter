package main

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rokoucha/akizuki-dg-route-b-exporter/MB_RL7023_11"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property/smartmeter"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/homeassistant"
)

const (
	echonetLitePort = 0x0E1A
	echonetLiteUDP  = 0x01

	responseTimeout = 20 * time.Second
	// 連続でこの回数応答が無ければセッションが切れたとみなして再接続する
	maxPollFailures = 3

	joinBackoffMin = 1 * time.Minute
	joinBackoffMax = 30 * time.Minute

	// ARIB STD-T108 の 1 時間あたりの送信時間総和の上限
	ARIBTransmissionLimit = 360 * time.Second
)

var (
	ErrNoResponse = errors.New("no response")

	controllerEOJ = [3]uint8{0x05, 0xff, 0x01}
	smartMeterEOJ = [3]uint8{smartmeter.ClassGroupCode, smartmeter.ClassCode, 0x01}
)

type txSample struct {
	at  time.Time
	sum time.Duration
}

type meter struct {
	addr     string
	channel  string
	panID    string
	interval time.Duration
	txBudget time.Duration

	ha     *homeassistant.Client
	logger *slog.Logger
	mb     *MB_RL7023_11.MB_RL7023_11

	// シリアルのリスナーから更新される
	rateLimited atomic.Bool
	sessionLost atomic.Bool
	mu          sync.Mutex
	pending     map[[2]uint8]chan *echonetlite.Frame

	joined      bool
	tid         uint16
	coefficient float64
	unit        float64
	txSamples   []txSample
}

// シリアルから流れてくる非同期イベントを処理します。
func (m *meter) listener(lines []string) error {
	for _, event := range MB_RL7023_11.ParseEvent(lines) {
		switch e := event.(type) {
		case *MB_RL7023_11.EVENT:
			switch e.Num {
			case MB_RL7023_11.EVENTNumTransmissionRateLimitExceeded:
				m.logger.Warn("ARIB STD-T108 transmission time limit exceeded, transmissions are suspended")
				m.rateLimited.Store(true)
			case MB_RL7023_11.EVENTNumTransmissionRateLimitReleased:
				m.logger.Info("ARIB STD-T108 transmission time limit released")
				m.rateLimited.Store(false)
			case MB_RL7023_11.EVENTNumSessionCloseRequestReceived,
				MB_RL7023_11.EVENTNumPANASessionClosed,
				MB_RL7023_11.EVENTNumPANASessionTimeout:
				m.logger.Warn("PANA session lost", "event", e.Num)
				m.sessionLost.Store(true)
			}

		case *MB_RL7023_11.ERXUDP:
			if !m.mb.IsResponse(e, echonetLiteUDP, m.addr, echonetLitePort) {
				continue
			}
			f, err := echonetlite.NewFrame(e.Data)
			if err != nil {
				m.logger.Debug("Failed to parse ECHONET Lite frame", "err", err)
				continue
			}
			m.mu.Lock()
			ch, ok := m.pending[f.TID]
			m.mu.Unlock()
			if !ok {
				m.logger.Debug("Received unsolicited frame", "frame", f)
				continue
			}
			select {
			case ch <- f:
			default:
			}
		}
	}
	return nil
}

func (m *meter) Run(ctx context.Context) error {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	backoff := joinBackoffMin
	failures := 0
	for {
		if m.sessionLost.Swap(false) && m.joined {
			m.disconnected()
		}

		if !m.joined {
			if err := m.join(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// 再接続のたびに送信するので、間隔を伸ばして送信時間を浪費しないようにする
				m.logger.Warn("Failed to join PAN, retrying", "err", err, "after", backoff)
				if err := sleep(ctx, backoff); err != nil {
					return err
				}
				backoff = min(backoff*2, joinBackoffMax)
				continue
			}
			backoff = joinBackoffMin
			failures = 0
		}

		if err := m.poll(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			failures++
			m.logger.Warn("Failed to poll smart meter", "err", err, "failures", failures)
			if failures >= maxPollFailures {
				m.disconnected()
			}
		} else {
			failures = 0
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *meter) disconnected() {
	m.joined = false
	if m.ha != nil {
		m.ha.SetAvailable(false)
	}
}

func (m *meter) join(ctx context.Context) error {
	if ok, err := m.canTransmit(ctx); err != nil || !ok {
		return errors.Join(MB_RL7023_11.ErrTransmissionRateLimited, err)
	}

	if _, err := m.mb.SKSREG(ctx, MB_RL7023_11.RegisterChannel, m.channel); err != nil {
		return err
	}
	if _, err := m.mb.SKSREG(ctx, MB_RL7023_11.RegisterPANID, m.panID); err != nil {
		return err
	}

	m.logger.Info("Joining to PAN", "addr", m.addr)
	if err := m.mb.SKJOIN(ctx, m.addr); err != nil {
		return err
	}
	m.logger.Info("Joined to PAN", "addr", m.addr)

	m.joined = true
	m.sessionLost.Store(false)
	if m.ha != nil {
		m.ha.SetAvailable(true)
	}
	return nil
}

func (m *meter) Close() {
	if !m.joined {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := m.mb.SKTERM(ctx); err != nil {
		m.logger.Error("Failed to execute command: SKTERM", "err", err)
		return
	}
	m.joined = false
	m.logger.Info("Disconnected from", "addr", m.addr)
}

// 送信してよいかを判定します。
// モジュール自体も ARIB STD-T108 の上限で送信を止めるが、上限に達すると最大 1 時間通信できなくなるので
// 直近 1 時間の送信時間が予算を超えたら自主的に送信を控える。
func (m *meter) canTransmit(ctx context.Context) (bool, error) {
	used, err := m.transmissionTimeLastHour(ctx)
	if err != nil {
		return false, err
	}
	limited, err := m.mb.IsTransmissionRateLimited(ctx)
	if err != nil {
		return false, err
	}
	m.rateLimited.Store(limited)

	if m.ha != nil {
		m.ha.Publish("transmission_time", used.Seconds())
		m.ha.Publish("transmission_rate_limited", limited)
	}

	if limited {
		m.logger.Warn("Skip transmission: module is rate limited by ARIB STD-T108", "used", used)
		return false, nil
	}
	if used >= m.txBudget {
		m.logger.Warn("Skip transmission: transmission time budget exceeded", "used", used, "budget", m.txBudget)
		return false, nil
	}
	m.logger.Debug("Transmission time", "used", used, "budget", m.txBudget)
	return true, nil
}

// 無線送信の積算時間 (SFD) を記録して直近 1 時間の送信時間を返します。
// 起動直後は起動時からの値になるが、それ以前の分はモジュールの制限フラグで補う。
func (m *meter) transmissionTimeLastHour(ctx context.Context) (time.Duration, error) {
	sum, err := m.mb.TransmissionSumTime(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()

	// リセット等で積算値が戻った
	if len(m.txSamples) > 0 && sum < m.txSamples[len(m.txSamples)-1].sum {
		m.txSamples = nil
	}
	m.txSamples = append(m.txSamples, txSample{at: now, sum: sum})

	// 1 時間以上前のサンプルのうち最新のものを基準として残す (多めに見積もる側に倒す)
	for len(m.txSamples) > 1 && now.Sub(m.txSamples[1].at) >= time.Hour {
		m.txSamples = m.txSamples[1:]
	}

	return sum - m.txSamples[0].sum, nil
}

// スマートメーターにプロパティ値読み出し要求を送り、応答を待ちます。
func (m *meter) get(ctx context.Context, props []property.Property) (*echonetlite.Frame, error) {
	m.tid++
	tid := [2]uint8{uint8(m.tid >> 8), uint8(m.tid)}

	frame := &echonetlite.Frame{
		EHD1: echonetlite.EHD1ECHONETLite,
		EHD2: echonetlite.EHD2SpecifiedMessageFormat,
		TID:  tid,
		EDATA: echonetlite.Data{
			SEOJ:       controllerEOJ,
			DEOJ:       smartMeterEOJ,
			ESV:        echonetlite.ESVGet,
			Properties: props,
		},
	}

	ch := make(chan *echonetlite.Frame, 1)
	m.mu.Lock()
	m.pending[tid] = ch
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.pending, tid)
		m.mu.Unlock()
	}()

	err := m.mb.SKSENDTO(ctx, echonetLiteUDP, m.addr, echonetLitePort, MB_RL7023_11.SKSENDTOSecStrict, MB_RL7023_11.SKSENDTOReservedValue, frame.Bytes())
	if err != nil {
		if errors.Is(err, MB_RL7023_11.ErrTransmissionRateLimited) {
			m.rateLimited.Store(true)
		}
		return nil, err
	}

	timer := time.NewTimer(responseTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, ErrNoResponse
		case f := <-ch:
			if f.EDATA.SEOJ != smartMeterEOJ || (f.EDATA.ESV != echonetlite.ESVGet_Res && f.EDATA.ESV != echonetlite.ESVGet_SNA) {
				continue
			}
			return f, nil
		}
	}
}

func (m *meter) poll(ctx context.Context) error {
	if ok, err := m.canTransmit(ctx); err != nil || !ok {
		return err
	}

	props := []property.Property{
		&smartmeter.MeasuredInstantaneousElectricPower{},
		&smartmeter.MeasuredInstantaneousCurrents{},
		&smartmeter.MeasuredCumulativeAmountOfElectricEnergyNormalDirection{},
		&smartmeter.MeasuredCumulativeAmountOfElectricEnergyReverseDirection{},
	}
	// 積算電力量の換算に必要な値は変わらないので、取れるまで同じ要求に相乗りさせる
	if m.unit == 0 || m.coefficient == 0 {
		props = append(props,
			&smartmeter.Coefficient{},
			&smartmeter.UnitForCumulativeAmountOfElectricEnergy{},
		)
	}

	f, err := m.get(ctx, props)
	if err != nil {
		return err
	}

	values := map[string]any{}
	var normal, reverse *uint32
	for _, p := range f.EDATA.Properties {
		switch v := p.(type) {
		case *smartmeter.MeasuredInstantaneousElectricPower:
			values["power"] = v.Value
		case *smartmeter.MeasuredInstantaneousCurrents:
			values["current_r"] = v.R
			values["current_t"] = v.T
		case *smartmeter.MeasuredCumulativeAmountOfElectricEnergyNormalDirection:
			normal = &v.Value
		case *smartmeter.MeasuredCumulativeAmountOfElectricEnergyReverseDirection:
			reverse = &v.Value
		case *smartmeter.Coefficient:
			m.coefficient = float64(v.Value)
		case *smartmeter.UnitForCumulativeAmountOfElectricEnergy:
			m.unit = v.Value
		case *property.UnknownProperty:
			// 係数 (D3) は任意プロパティで、無い場合は 1 とみなす
			if v.EPC == smartmeter.EPCCoefficient {
				m.coefficient = 1
			}
		}
	}

	if m.unit != 0 && m.coefficient != 0 {
		if normal != nil {
			values["energy_normal"] = m.kWh(*normal)
		}
		if reverse != nil {
			values["energy_reverse"] = m.kWh(*reverse)
		}
	}

	m.logger.Info("Measured", "values", values)
	if m.ha != nil {
		for _, s := range homeassistant.Sensors {
			if v, ok := values[s.Key]; ok {
				m.ha.Publish(s.Key, v)
			}
		}
	}

	return nil
}

// 積算電力量計測値を kWh に換算します。単位は最小 0.0001kWh なので浮動小数点の誤差を丸める
func (m *meter) kWh(v uint32) float64 {
	return math.Round(float64(v)*m.coefficient*m.unit*10000) / 10000
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
