package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/MB_RL7023_11"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/homeassistant"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/serial"
)

// スマートメーターの計測値は 1 分程度で更新されるので、それより短い間隔で問い合わせても電波を浪費するだけ
const minPollInterval = 10 * time.Second

const version = "1.0.1"

type options struct {
	BaudRate uint `short:"b" long:"baud-rate" description:"Baud rate to connect to Wi-SUN module" default:"115200"`
	Scan     bool `short:"s" long:"scan" description:"Scan for available PANs"`
	Verbose  bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	Version  bool `short:"V" long:"version" description:"Show version"`

	RouteBID       string `long:"route-b-id" env:"ROUTE_B_ID" description:"Route B ID" required:"true"`
	RouteBPassword string `long:"route-b-password" env:"ROUTE_B_PASSWORD" description:"Route B password" required:"true"`
	Channel        string `long:"channel" env:"ROUTE_B_CHANNEL" description:"Channel of smart meter (from --scan)"`
	PANID          string `long:"pan-id" env:"ROUTE_B_PANID" description:"PAN ID of smart meter (from --scan)"`
	Addr           string `long:"addr" env:"ROUTE_B_ADDR" description:"IPv6 address of smart meter (from --scan)"`

	PollInterval time.Duration `long:"poll-interval" env:"POLL_INTERVAL" description:"Interval to poll smart meter" default:"60s"`
	TxBudget     time.Duration `long:"tx-budget" env:"TX_BUDGET" description:"Self-imposed transmission time budget per hour (ARIB STD-T108 limit is 360s)" default:"180s"`

	MQTTBroker          string `long:"mqtt-broker" env:"MQTT_BROKER" description:"MQTT broker URL for Home Assistant (e.g. tcp://homeassistant.local:1883)"`
	MQTTUsername        string `long:"mqtt-username" env:"MQTT_USERNAME" description:"MQTT username"`
	MQTTPassword        string `long:"mqtt-password" env:"MQTT_PASSWORD" description:"MQTT password"`
	MQTTDiscoveryPrefix string `long:"mqtt-discovery-prefix" env:"MQTT_DISCOVERY_PREFIX" description:"Home Assistant MQTT discovery prefix" default:"homeassistant"`

	Args struct {
		Port string `positional-arg-name:"serial-port" required:"true"`
	} `positional-args:"true"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --version だけで実行できるよう、必須オプションの検証より前に見る
	for _, arg := range os.Args[1:] {
		if arg == "-V" || arg == "--version" {
			fmt.Println(version)
			return
		}
	}

	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	logLevel := slog.LevelInfo
	if opts.Verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	if err := run(ctx, opts, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Exiting", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options, logger *slog.Logger) error {
	if !opts.Scan {
		if opts.Channel == "" || opts.PANID == "" || opts.Addr == "" {
			return errors.New("please set ROUTE_B_CHANNEL, ROUTE_B_PANID and ROUTE_B_ADDR (run with --scan to find them)")
		}
		if opts.PollInterval < minPollInterval {
			return fmt.Errorf("poll interval must be at least %s", minPollInterval)
		}
		if opts.TxBudget <= 0 || opts.TxBudget > ARIBTransmissionLimit {
			return fmt.Errorf("transmission time budget must be in (0, %s]", ARIBTransmissionLimit)
		}
	}

	s, err := serial.New(serial.Config{
		PortName: opts.Args.Port,
		BaudRate: int(opts.BaudRate),
	})
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}
	defer s.Close()

	// 終了時に SKTERM を送れるよう、シリアルの読み出しはシグナルとは別に止める
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	ready := make(chan struct{})
	errCh := make(chan error, 1)
	go s.Streaming(streamCtx, ready, errCh)
	<-ready

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	go func() {
		select {
		case err := <-errCh:
			cancel(fmt.Errorf("serial: %w", err))
		case <-streamCtx.Done():
		}
	}()

	debugListener := func(lines []string) error {
		for _, line := range lines {
			logger.Debug("streaming", "line", line)
		}
		return nil
	}
	s.AddListner(&debugListener)

	mb := MB_RL7023_11.New(MB_RL7023_11.Config{
		Logger: logger,
		Serial: s,
	})
	if err := mb.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize Wi-SUN module: %w", err)
	}

	ver, err := mb.SKVER(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute command: SKVER: %w", err)
	}
	info, err := mb.SKINFO(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute command: SKINFO: %w", err)
	}
	logger.Info("Wi-SUN module info", "version", ver, "info", info, "app_version", version)

	if err := mb.SKSETRBID(ctx, opts.RouteBID); err != nil {
		return fmt.Errorf("failed to execute command: SKSETRBID: %w", err)
	}
	if err := mb.SKSETPWD(ctx, opts.RouteBPassword); err != nil {
		return fmt.Errorf("failed to execute command: SKSETPWD: %w", err)
	}

	if opts.Scan {
		return scan(ctx, mb, logger)
	}

	m := &meter{
		addr:     opts.Addr,
		channel:  opts.Channel,
		panID:    opts.PANID,
		interval: opts.PollInterval,
		txBudget: opts.TxBudget,
		logger:   logger,
		mb:       mb,
		pending:  map[[2]uint8]chan *echonetlite.Frame{},
	}

	if opts.MQTTBroker != "" {
		m.ha = homeassistant.New(homeassistant.Config{
			Broker:          opts.MQTTBroker,
			Username:        opts.MQTTUsername,
			Password:        opts.MQTTPassword,
			DiscoveryPrefix: opts.MQTTDiscoveryPrefix,
			NodeID:          info.Addr64,
			Version:         version,
			Logger:          logger,
		})
		defer m.ha.Close()
	}

	meterListener := m.listener
	s.AddListner(&meterListener)

	err = m.Run(ctx)
	m.Close()
	if cause := context.Cause(ctx); cause != nil && !errors.Is(cause, context.Canceled) {
		return cause
	}
	return err
}

func scan(ctx context.Context, mb *MB_RL7023_11.MB_RL7023_11, logger *slog.Logger) error {
	res, err := mb.SKSCAN(ctx, MB_RL7023_11.SKSCANModeActiveWithIE, 0xFFFFFFFF, 6, MB_RL7023_11.SKSCANReservedValue)
	if err != nil {
		return fmt.Errorf("failed to execute command: SKSCAN: %w", err)
	}
	if len(res) == 0 {
		logger.Info("No PANs found")
		return nil
	}

	logger.Info("Found PANs")
	for i, v := range res {
		pan := v.(*MB_RL7023_11.EPANDESC)
		ipaddr, err := mb.SKLL64(ctx, pan.Addr)
		if err != nil {
			return fmt.Errorf("failed to execute command: SKLL64: %w", err)
		}

		fmt.Printf("%d:\n  Channel: %X\n  ChannelPage: %X\n  PanID: %X\n  Addr: %s\n  LQI: %X\n  Side: %X\n  PairID: %s\n\n",
			i+1,
			pan.Channel,
			pan.ChannelPage,
			pan.PanID,
			ipaddr,
			pan.LQI,
			pan.Side,
			pan.PairID,
		)
	}
	return nil
}
