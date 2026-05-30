package wecom

import (
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/busadapter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("wecom_event_bus_unavailable")

// BusAdapter mirrors WeCom-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter = busadapter.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "wecom".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return busadapter.New(dispatcher, "wecom", wecomBusTraceID, ErrEventBusUnavailable)
}

func wecomBusTraceID(ev gateway.InboundEvent) string {
	return busadapter.TraceIDFromChatUserMessage("wecom", ev)
}
