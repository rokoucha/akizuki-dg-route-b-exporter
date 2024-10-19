package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/MB_RL7023_11"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/serial"
)

type options struct {
	BaudRate *uint `short:"b" long:"baud-rate" description:"Baud rate to connect to Wi-SUN module, default: 115200"`
	Scan     *bool `short:"s" long:"scan" description:"Scan for available PANs"`
	Verbose  *bool `short:"v" long:"verbose" description:"Show verbose debug information"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var opts options
	args, err := flags.Parse(&opts)
	if err != nil {
		flagsErr := err.(*flags.Error)
		if flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	logLevel := slog.LevelInfo
	if opts.Verbose != nil && *opts.Verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	if len(args) != 1 {
		logger.Error("Please specify a serial port")
		os.Exit(1)
	}
	portName := args[0]

	var baudrate uint
	if opts.BaudRate != nil {
		baudrate = *opts.BaudRate
	} else {
		baudrate = 115200
	}

	var scanMode bool
	if opts.Scan != nil {
		scanMode = *opts.Scan
	} else {
		scanMode = false
	}

	serial, err := serial.New(serial.Config{
		PortName: portName,
		BaudRate: int(baudrate),
	})
	if err != nil {
		logger.Error("Failed to open serial port", "err", err)
		os.Exit(1)
	}
	closer := func() {
		serial.Close()
	}
	defer func() {
		closer()
	}()
	go func() {
		<-ctx.Done()
		closer()
	}()

	listener := func(lines []string) error {
		for _, line := range lines {
			logger.Debug("streaming", "line", line)
		}
		events := MB_RL7023_11.ParseEvent(lines)
		for _, event := range events {
			if e, ok := event.(*MB_RL7023_11.ERXUDP); ok {
				if d, ok := e.Data.(*MB_RL7023_11.ECHONETLiteFrame); ok {
					logger.Info("Received UDP packet", "event", e, "data", d)
				}
			}
		}
		return nil
	}

	serial.AddListner(&listener)

	ready := make(chan struct{})
	errCh := make(chan error)
	go serial.Streaming(ctx, ready, errCh)
	<-ready

	mb := MB_RL7023_11.New(MB_RL7023_11.Config{
		Logger: logger,
		Serial: serial,
	})
	err = mb.Initialize(ctx)
	if err != nil {
		logger.Error("Failed to initialize Wi-SUN module", "err", err)
		os.Exit(1)
	}

	ver, err := mb.SKVER(ctx)
	if err != nil {
		logger.Error("Failed to execute command: SKVER", "err", err)
		os.Exit(1)
	}
	info, err := mb.SKINFO(ctx)
	if err != nil {
		logger.Error("Failed to execute command: SKINFO", "err", err)
		os.Exit(1)
	}
	logger.Info("Wi-SUN module info", "version", ver, "info", info)

	rbid := os.Getenv("ROUTE_B_ID")
	if rbid == "" {
		logger.Error("Please set ROUTE_B_ID env variable")
		os.Exit(1)
	}

	err = mb.SKSETRBID(ctx, rbid)
	if err != nil {
		logger.Error("Failed to execute command: SKSETRBID", "err", err)
		os.Exit(1)
	}

	rbpwd := os.Getenv("ROUTE_B_PASSWORD")
	if rbpwd == "" {
		logger.Error("Please set ROUTE_B_PASSWORD env variable")
		os.Exit(1)
	}

	err = mb.SKSETPWD(ctx, rbpwd)
	if err != nil {
		logger.Error("Failed to execute command: SKSETPWD", "err", err)
		os.Exit(1)
	}

	if scanMode {
		res, err := mb.SKSCAN(ctx, MB_RL7023_11.SKSCANModeActiveWithIE, 0xFFFFFFFF, 6, MB_RL7023_11.SKSCANReservedValue)
		if err != nil {
			logger.Error("Failed to execute command: SKSCAN", "err", err)
			os.Exit(1)
		}
		var pans []*MB_RL7023_11.EPANDESC
		for _, pan := range res {
			pans = append(pans, pan.(*MB_RL7023_11.EPANDESC))
		}
		if len(pans) == 0 {
			logger.Info("No PANs found")
			os.Exit(0)
		}
		logger.Info("Found PANs")
		for i, pan := range pans {
			ipaddr, err := mb.SKLL64(ctx, pan.Addr)
			if err != nil {
				logger.Error("Failed to execute command: SKLL64", "err", err)
				os.Exit(1)
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
		os.Exit(0)
	}

	channel := os.Getenv("ROUTE_B_CHANNEL")
	if channel == "" {
		logger.Error("Please set ROUTE_B_CHANNEL env variable")
		os.Exit(1)
	}

	_, err = mb.SKSREG(ctx, MB_RL7023_11.RegisterChannel, channel)
	if err != nil {
		logger.Error("Failed to execute command: SKSREG", "err", err)
		os.Exit(1)
	}

	panid := os.Getenv("ROUTE_B_PANID")
	if panid == "" {
		logger.Error("Please set ROUTE_B_PANID env variable")
		os.Exit(1)
	}

	_, err = mb.SKSREG(ctx, MB_RL7023_11.RegisterPANID, panid)
	if err != nil {
		logger.Error("Failed to execute command: SKSREG", "err", err)
		os.Exit(1)
	}

	addr := os.Getenv("ROUTE_B_ADDR")
	if addr == "" {
		logger.Error("Please set ROUTE_B_ADDR env variable")
		os.Exit(1)
	}

	logger.Info("Joining to PAN", "addr", addr)
	err = mb.SKJOIN(ctx, addr)
	if err != nil {
		logger.Error("Failed to execute command: SKJOIN", "err", err)
		os.Exit(1)
	}

	closer = func() {
		ctx := context.Background()
		err = mb.SKTERM(ctx)
		if err != nil {
			logger.Error("Failed to execute command: SKTERM", "err", err)
			os.Exit(1)
		}
		logger.Info("Disconnected from", "addr", addr)

		serial.Close()
	}

	for {
		// logger.Info("Send command frame")
		// frame := &MB_RL7023_11.ECHONETLiteFrame{
		// 	EHD1: MB_RL7023_11.ECHONETLiteEHD1ECHONETLite,
		// 	EHD2: MB_RL7023_11.ECHONETLiteEHD2SpecifiedMessageFormat,
		// 	TID:  [2]uint8{0x00, 0x01},
		// 	EDATA: MB_RL7023_11.ECHONETLiteData{
		// 		SEOJ: [3]uint8{0x05, 0xff, 0x01},
		// 		DEOJ: [3]uint8{0x02, 0x88, 0x01},
		// 		ESV:  MB_RL7023_11.ECHONETLiteESVGet,
		// 		Props: []MB_RL7023_11.ECHONETPropertySet{
		// 			{
		// 				EPC: 0xe7,
		// 				EDT: []uint8{},
		// 			},
		// 		},
		// 	},
		// }
		// received, err := mb.SKSENDTO(ctx, 0x01, addr, 0x0E1A, MB_RL7023_11.SKSENDTOSecStrict, MB_RL7023_11.SKSENDTOReservedValue, frame)
		// if err != nil {
		// 	logger.Error("Failed to execute command: SKSENDTO", "err", err)
		// 	os.Exit(1)
		// }

		// kw, err := received.Data.InstantaneousPowerMeasurementValue()
		// if err == nil {
		// 	logger.Info("Instantaneous power measurement value", "kw", kw)
		// } else {
		// 	logger.Error("Failed to parse packet", "err", err)
		// }

		time.Sleep(60 * time.Second)
	}
}
