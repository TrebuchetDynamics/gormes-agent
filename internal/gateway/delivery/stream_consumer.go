package delivery

import (
	deliverystream "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/stream"
)

// StreamDeliverySink is the minimal contract for replaying one render frame to
// one resolved destination.
type StreamDeliverySink = deliverystream.StreamDeliverySink

// DeliveryResult captures the outcome for one attempted fan-out target.
type DeliveryResult = deliverystream.DeliveryResult

// StreamConsumer fans one kernel frame out to one or more delivery targets in
// a deterministic order.
type StreamConsumer = deliverystream.StreamConsumer

func NewStreamConsumer(sink StreamDeliverySink) *StreamConsumer {
	return deliverystream.NewStreamConsumer(sink)
}
