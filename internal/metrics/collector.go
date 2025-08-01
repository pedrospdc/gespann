package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pedrospdc/gespann/internal/adapters"
	"github.com/pedrospdc/gespann/pkg/types"
)

type Collector struct {
	adapters []adapters.MetricsAdapter
	metrics  types.ConnMetrics
	mutex    sync.RWMutex
	logger   *slog.Logger
}

func NewCollector(adapters []adapters.MetricsAdapter, logger *slog.Logger) *Collector {
	return &Collector{
		adapters: adapters,
		logger:   logger,
	}
}

func (c *Collector) ProcessEvent(event types.ConnEvent) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch event.Type {
	case types.ConnOpen:
		c.metrics.OpenConnections++
		c.metrics.TotalConnections++
		switch event.Protocol {
		case types.ProtoTCP:
			c.metrics.TCPConnections++
		case types.ProtoUDP:
			c.metrics.UDPConnections++
		}
	case types.ConnClose:
		c.metrics.OpenConnections--
		c.metrics.ClosedConnections++
		c.updatePerformanceMetrics(event)
	case types.ConnReset:
		c.metrics.OpenConnections--
		c.metrics.ResetConnections++
		c.updatePerformanceMetrics(event)
	case types.ConnFailed:
		c.metrics.FailedConnections++
	case types.ConnIdle:
		c.metrics.IdleConnections++
	}

	for _, adapter := range c.adapters {
		if err := adapter.SendEvent(context.Background(), event); err != nil {
			c.logger.Error("failed to send event to adapter", "error", err)
		}
	}
}

func (c *Collector) updatePerformanceMetrics(event types.ConnEvent) {
	c.metrics.TotalBytesSent += event.BytesSent
	c.metrics.TotalBytesReceived += event.BytesReceived

	// Simple moving average for RTT and duration
	if event.RTTMicros > 0 {
		c.metrics.AvgRTT = (c.metrics.AvgRTT + float64(event.RTTMicros)) / 2.0
	}
	if event.DurationMS > 0 {
		c.metrics.AvgConnectionDuration = (c.metrics.AvgConnectionDuration + float64(event.DurationMS)) / 2.0
	}
}

func (c *Collector) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mutex.RLock()
			currentMetrics := c.metrics
			c.mutex.RUnlock()

			for _, adapter := range c.adapters {
				if err := adapter.SendMetrics(ctx, currentMetrics); err != nil {
					c.logger.Error("failed to send metrics to adapter", "error", err)
				}
			}
		}
	}
}

func (c *Collector) Close() error {
	var errs []error
	for _, adapter := range c.adapters {
		if err := adapter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		c.logger.Error("errors closing adapters", "errors", errs)
		return errs[0]
	}

	return nil
}
