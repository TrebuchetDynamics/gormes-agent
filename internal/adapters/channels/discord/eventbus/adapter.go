package eventbus

import (
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/busadapter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("discord_event_bus_unavailable")

// BusAdapter mirrors Discord-normalized inbound events onto the shared
// gateway event bus without changing the existing channel inbox path.
type BusAdapter = busadapter.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "discord".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return busadapter.New(dispatcher, "discord", DiscordBusTraceID, ErrEventBusUnavailable)
}

func DiscordBusTraceID(ev gateway.InboundEvent) string {
	return busadapter.TraceIDFromGuildChatThreadMessage("discord", ev)
}
