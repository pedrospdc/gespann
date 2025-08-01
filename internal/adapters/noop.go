package adapters

import (
	"context"

	"github.com/pedrospdc/gespann/pkg/types"
)

type NoOpAdapter struct{}

func NewNoOpAdapter() *NoOpAdapter {
	return &NoOpAdapter{}
}

func (n *NoOpAdapter) SendMetrics(ctx context.Context, metrics types.ConnMetrics) error {
	return nil
}

func (n *NoOpAdapter) SendEvent(ctx context.Context, event types.ConnEvent) error {
	return nil
}

func (n *NoOpAdapter) Close() error {
	return nil
}
