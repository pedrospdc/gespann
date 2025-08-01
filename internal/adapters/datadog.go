package adapters

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/pedrospdc/gespann/pkg/types"
)

type DataDogAdapter struct {
	client *statsd.Client
}

func NewDataDogAdapter(settings map[string]string) (*DataDogAdapter, error) {
	host := settings["host"]
	if host == "" {
		host = "localhost:8125"
	}

	client, err := statsd.New(host)
	if err != nil {
		return nil, fmt.Errorf("failed to create DataDog client: %w", err)
	}

	return &DataDogAdapter{
		client: client,
	}, nil
}

func (d *DataDogAdapter) SendMetrics(ctx context.Context, metrics types.ConnMetrics) error {
	if err := d.client.Gauge("gespann.open_connections", float64(metrics.OpenConnections), nil, 1); err != nil {
		return err
	}

	if err := d.client.Count("gespann.closed_connections", metrics.ClosedConnections, nil, 1); err != nil {
		return err
	}

	if err := d.client.Gauge("gespann.idle_connections", float64(metrics.IdleConnections), nil, 1); err != nil {
		return err
	}

	if err := d.client.Count("gespann.total_connections", metrics.TotalConnections, nil, 1); err != nil {
		return err
	}

	return nil
}

func (d *DataDogAdapter) SendEvent(ctx context.Context, event types.ConnEvent) error {
	eventType := ""
	switch event.Type {
	case types.ConnOpen:
		eventType = "open"
	case types.ConnClose:
		eventType = "close"
	case types.ConnIdle:
		eventType = "idle"
	}

	tags := []string{
		"event_type:" + eventType,
		"pid:" + strconv.FormatUint(uint64(event.PID), 10),
		fmt.Sprintf("src:%d.%d.%d.%d:%d",
			event.SAddr&0xFF, (event.SAddr>>8)&0xFF,
			(event.SAddr>>16)&0xFF, (event.SAddr>>24)&0xFF,
			event.SPort),
		fmt.Sprintf("dst:%d.%d.%d.%d:%d",
			event.DAddr&0xFF, (event.DAddr>>8)&0xFF,
			(event.DAddr>>16)&0xFF, (event.DAddr>>24)&0xFF,
			event.DPort),
	}

	return d.client.Incr("gespann.connection_events", tags, 1)
}

func (d *DataDogAdapter) Close() error {
	return d.client.Close()
}
