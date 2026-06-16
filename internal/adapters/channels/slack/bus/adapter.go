package bus

import (
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/busadapter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("slack_event_bus_unavailable")

// BusAdapter mirrors Slack-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter = busadapter.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "slack".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return busadapter.New(dispatcher, "slack", TraceID, ErrEventBusUnavailable)
}

func TraceID(ev gateway.InboundEvent) string {
	return busadapter.TraceIDFromAccountChatThreadMessage("slack", ev)
}
