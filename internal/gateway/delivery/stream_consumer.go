package delivery

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// StreamDeliverySink is the minimal contract for replaying one render frame to
// one resolved destination.
type StreamDeliverySink interface {
	DeliverFrame(ctx context.Context, target Target, frame kernel.RenderFrame) error
}

// DeliveryResult captures the outcome for one attempted fan-out target.
type DeliveryResult struct {
	Target Target
	Err    error
}

// StreamConsumer fans one kernel frame out to one or more delivery targets in
// a deterministic order.
type StreamConsumer struct {
	sink StreamDeliverySink
}

func NewStreamConsumer(sink StreamDeliverySink) *StreamConsumer {
	return &StreamConsumer{sink: sink}
}

func (c *StreamConsumer) FanOut(ctx context.Context, frame kernel.RenderFrame, targets []Target) []DeliveryResult {
	results := make([]DeliveryResult, 0, len(targets))
	if c == nil || c.sink == nil {
		return results
	}
	for _, target := range targets {
		err := c.sink.DeliverFrame(ctx, target, frame)
		results = append(results, DeliveryResult{Target: target, Err: err})
	}
	return results
}
