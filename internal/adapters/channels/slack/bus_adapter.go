package slack

import (
	slackbus "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/slack/bus"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = slackbus.ErrEventBusUnavailable

// BusAdapter mirrors Slack-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter = slackbus.BusAdapter

// NewBusAdapter returns a BusAdapter that publishes events tagged with "slack".
func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return slackbus.NewBusAdapter(dispatcher)
}

func slackBusTraceID(ev gateway.InboundEvent) string {
	return slackbus.TraceID(ev)
}
