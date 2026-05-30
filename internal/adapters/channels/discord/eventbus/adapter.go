package eventbus

import (
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("discord_event_bus_unavailable")

// BusAdapter mirrors Discord-normalized inbound events onto the shared gateway
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
		ev.Platform = "discord"
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = DiscordBusTraceID(ev)
	}
	payload := gateway.MessageEventPayloadFromInbound(ev)
	return a.dispatcher.PublishMessageReceived("discord", traceID, payload)
}

func DiscordBusTraceID(ev gateway.InboundEvent) string {
	parts := []string{"discord"}
	if guildID := strings.TrimSpace(ev.GuildID); guildID != "" {
		parts = append(parts, guildID)
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
