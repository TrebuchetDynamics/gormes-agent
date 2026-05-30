package discord

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/eventbus"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = eventbus.ErrEventBusUnavailable

// BusAdapter mirrors Discord-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter = eventbus.BusAdapter

func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return eventbus.NewBusAdapter(dispatcher)
}

func discordBusTraceID(ev gateway.InboundEvent) string {
	return eventbus.DiscordBusTraceID(ev)
}
