package adapters

import (
	"context"

	"github.com/pedrospdc/gespann/pkg/types"
)

type MetricsAdapter interface {
	SendMetrics(ctx context.Context, metrics types.ConnMetrics) error
	SendEvent(ctx context.Context, event types.ConnEvent) error
	Close() error
}

type Config struct {
	Type     string            `yaml:"type"`
	Settings map[string]string `yaml:"settings"`
}

func NewAdapter(config Config) (MetricsAdapter, error) {
	switch config.Type {
	case "prometheus":
		return NewPrometheusAdapter(config.Settings)
	case "datadog":
		return NewDataDogAdapter(config.Settings)
	default:
		return NewNoOpAdapter(), nil
	}
}
