package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pedrospdc/gespann/internal/adapters"
	"github.com/pedrospdc/gespann/internal/config"
	"github.com/pedrospdc/gespann/internal/ebpf"
	"github.com/pedrospdc/gespann/internal/metrics"
	"github.com/pedrospdc/gespann/pkg/types"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = config.Load(configPath)
		if err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}
	} else {
		cfg = config.Default()
	}

	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	var adapterInstances []adapters.MetricsAdapter
	for _, adapterConfig := range cfg.Adapters {
		adapter, err := adapters.NewAdapter(adapterConfig)
		if err != nil {
			logger.Error("failed to create adapter", "type", adapterConfig.Type, "error", err)
			continue
		}
		adapterInstances = append(adapterInstances, adapter)
		logger.Info("adapter initialized", "type", adapterConfig.Type)
	}

	if len(adapterInstances) == 0 {
		logger.Error("no adapters configured")
		os.Exit(1)
	}

	collector := metrics.NewCollector(adapterInstances, logger)
	defer func() {
		if err := collector.Close(); err != nil {
			logger.Error("failed to close collector", "error", err)
		}
	}()

	tracker, err := ebpf.NewTracker(logger)
	if err != nil {
		logger.Error("failed to create eBPF tracker", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := tracker.Close(); err != nil {
			logger.Error("failed to close tracker", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := tracker.Start(ctx); err != nil {
		logger.Error("failed to start eBPF tracker", "error", err)
		os.Exit(1)
	}

	eventCh := make(chan types.ConnEvent, 1000)

	go func() {
		if err := tracker.ReadEvents(ctx, eventCh); err != nil && ctx.Err() == nil {
			logger.Error("error reading events", "error", err)
			cancel()
		}
	}()

	go func() {
		for event := range eventCh {
			collector.ProcessEvent(event)
		}
	}()

	go collector.Start(ctx, 10*time.Second)

	logger.Info("gespann started successfully")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Info("shutting down...")
	cancel()

	close(eventCh)
}
