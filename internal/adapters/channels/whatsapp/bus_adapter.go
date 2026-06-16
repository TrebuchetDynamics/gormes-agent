package whatsapp

import (
	whatsappbus "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/whatsapp/bus"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = whatsappbus.ErrEventBusUnavailable

// BusAdapter mirrors WhatsApp-normalized inbound events onto the shared
// gateway event bus without changing the existing channel inbox path.
type BusAdapter = whatsappbus.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "whatsapp".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return whatsappbus.NewBusAdapter(dispatcher)
}

func whatsappBusTraceID(ev gateway.InboundEvent) string {
	return whatsappbus.TraceID(ev)
}
