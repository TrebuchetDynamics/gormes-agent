package slack

import (
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("slack_event_bus_unavailable")

// BusAdapter mirrors Slack-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter struct {
	dispatcher *gateway.EventDispatcher
}

func NewBusAdapter(dispatcher *gateway.EventDispatcher) *BusAdapter {
	return &BusAdapter{dispatcher: dispatcher}
}

func (a *BusAdapter) PublishInboundMessage(traceID string, ev gateway.InboundEvent) error {
	if a == nil || a.dispatcher == nil || !a.dispatcher.Available() {
		return ErrEventBusUnavailable
	}
	if strings.TrimSpace(ev.Platform) == "" {
		ev.Platform = "slack"
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = slackBusTraceID(ev)
	}
	payload := gateway.MessageEventPayloadFromInbound(ev)
	return a.dispatcher.PublishMessageReceived("slack", traceID, payload)
}

func slackBusTraceID(ev gateway.InboundEvent) string {
	parts := []string{"slack"}
	if accountID := strings.TrimSpace(ev.AccountID); accountID != "" {
		parts = append(parts, accountID)
	}
	if chatID := strings.TrimSpace(ev.ChatID); chatID != "" {
		parts = append(parts, chatID)
	}
	if threadID := strings.TrimSpace(ev.ThreadID); threadID != "" {
		parts = append(parts, threadID)
	}
	if msgID := strings.TrimSpace(ev.MessageID); msgID != "" {
		parts = append(parts, msgID)
	} else if msgID := strings.TrimSpace(ev.MsgID); msgID != "" {
		parts = append(parts, msgID)
	}
	return strings.Join(parts, ":")
}
