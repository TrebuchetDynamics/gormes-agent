package gateway

import gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"

// StreamDeliverySink is the minimal contract for replaying one render frame to
// one resolved destination.
type StreamDeliverySink = gatewaydelivery.StreamDeliverySink

// DeliveryResult captures the outcome for one attempted fan-out target.
type DeliveryResult = gatewaydelivery.DeliveryResult

// GatewayStreamConsumer fans one kernel frame out to one or more delivery
// targets in a deterministic order.
type GatewayStreamConsumer = gatewaydelivery.StreamConsumer

func NewGatewayStreamConsumer(sink StreamDeliverySink) *GatewayStreamConsumer {
	return gatewaydelivery.NewStreamConsumer(sink)
}
