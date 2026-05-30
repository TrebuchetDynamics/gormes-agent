package whatsapp

import (
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/busadapter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("whatsapp_event_bus_unavailable")

// BusAdapter mirrors WhatsApp-normalized inbound events onto the shared
// gateway event bus without changing the existing channel inbox path.
type BusAdapter = busadapter.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "whatsapp".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return busadapter.New(dispatcher, platformName, whatsappBusTraceID, ErrEventBusUnavailable)
}

func whatsappBusTraceID(ev gateway.InboundEvent) string {
	return busadapter.TraceIDFromChatUserMessage(platformName, ev)
}
