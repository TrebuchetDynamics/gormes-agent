package weixin

import (
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/busadapter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("weixin_event_bus_unavailable")

// BusAdapter mirrors Weixin-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter = busadapter.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "weixin".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return busadapter.New(dispatcher, "weixin", weixinBusTraceID, ErrEventBusUnavailable)
}

func weixinBusTraceID(ev gateway.InboundEvent) string {
	return busadapter.TraceIDFromChatUserMessage("weixin", ev)
}
