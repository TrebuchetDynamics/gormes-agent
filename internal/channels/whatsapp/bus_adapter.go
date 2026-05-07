package whatsapp

import (
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var ErrEventBusUnavailable = errors.New("whatsapp_event_bus_unavailable")

// BusAdapter mirrors WhatsApp-normalized inbound events onto the shared
// gateway event bus without changing the existing channel inbox path.
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
		ev.Platform = platformName
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = whatsappBusTraceID(ev)
	}
	payload := gateway.MessageEventPayloadFromInbound(ev)
	return a.dispatcher.PublishMessageReceived(platformName, traceID, payload)
}

func whatsappBusTraceID(ev gateway.InboundEvent) string {
	parts := []string{platformName}
	if chatID := strings.TrimSpace(ev.ChatID); chatID != "" {
		parts = append(parts, chatID)
	}
	if userID := strings.TrimSpace(ev.UserID); userID != "" {
		parts = append(parts, userID)
	}
	if msgID := strings.TrimSpace(ev.MessageID); msgID != "" {
		parts = append(parts, msgID)
	} else if msgID := strings.TrimSpace(ev.MsgID); msgID != "" {
		parts = append(parts, msgID)
	}
	return strings.Join(parts, ":")
}
