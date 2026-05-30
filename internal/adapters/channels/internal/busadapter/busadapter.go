// Package busadapter provides a reusable BusAdapter implementation that
// mirrors channel-normalized inbound events onto the shared gateway event
// bus. Each channel creates a thin facade that passes its platform name,
// trace ID function, and unavailable error.
//
// This package lives under internal/ to prevent accidental public import.
package busadapter

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// BusAdapter mirrors channel-normalized inbound events onto the shared gateway
// event bus without changing the existing channel inbox path.
type BusAdapter struct {
	dispatcher    *gateway.EventDispatcher
	platformName  string
	traceIDFunc   func(gateway.InboundEvent) string
	errUnavailable error
}

// New returns a BusAdapter that publishes events tagged with platformName.
func New(dispatcher *gateway.EventDispatcher, platformName string, traceIDFunc func(gateway.InboundEvent) string, errUnavailable error) *BusAdapter {
	return &BusAdapter{
		dispatcher:    dispatcher,
		platformName:  platformName,
		traceIDFunc:   traceIDFunc,
		errUnavailable: errUnavailable,
	}
}

// PublishInboundMessage publishes a gateway.MessageEventPayload constructed
// from ev. If traceID is empty a platform-specific trace ID is auto-derived.
func (a *BusAdapter) PublishInboundMessage(traceID string, ev gateway.InboundEvent) error {
	if a == nil || a.dispatcher == nil || !a.dispatcher.Available() {
		return a.errUnavailable
	}
	if strings.TrimSpace(ev.Platform) == "" {
		ev.Platform = a.platformName
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = a.traceIDFunc(ev)
	}
	payload := gateway.MessageEventPayloadFromInbound(ev)
	return a.dispatcher.PublishMessageReceived(a.platformName, traceID, payload)
}

// PlatformName returns the platform name this adapter was configured for.
func (a *BusAdapter) PlatformName() string {
	return a.platformName
}

// TraceIDFromChatUserMessage builds a trace ID from platform:chatID:userID:messageID.
func TraceIDFromChatUserMessage(platform string, ev gateway.InboundEvent) string {
	parts := []string{platform}
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

// TraceIDFromAccountChatThreadMessage builds a trace ID from
// platform:accountID:chatID:threadID:messageID.
func TraceIDFromAccountChatThreadMessage(platform string, ev gateway.InboundEvent) string {
	parts := []string{platform}
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

// TraceIDFromChatMessage builds a trace ID from platform:chatID:messageID.
func TraceIDFromChatMessage(platform string, ev gateway.InboundEvent) string {
	parts := []string{platform}
	if chatID := strings.TrimSpace(ev.ChatID); chatID != "" {
		parts = append(parts, chatID)
	}
	if msgID := strings.TrimSpace(ev.MessageID); msgID != "" {
		parts = append(parts, msgID)
	} else if msgID := strings.TrimSpace(ev.MsgID); msgID != "" {
		parts = append(parts, msgID)
	}
	return strings.Join(parts, ":")
}

// TraceIDFromGuildChatThreadMessage builds a trace ID from
// platform:guildID:chatID:threadID:messageID. Used by Discord.
func TraceIDFromGuildChatThreadMessage(platform string, ev gateway.InboundEvent) string {
	parts := []string{platform}
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
