package telegram

import (
	telegrambus "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/bus"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = telegrambus.ErrEventBusUnavailable

// BusAdapter mirrors Telegram-normalized inbound events onto the shared
// gateway event bus without changing the existing channel inbox path.
type BusAdapter = telegrambus.BusAdapter

func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return telegrambus.NewBusAdapter(dispatcher)
}

func telegramBusTraceID(ev gateway.InboundEvent) string {
	return telegrambus.TraceID(ev)
}
