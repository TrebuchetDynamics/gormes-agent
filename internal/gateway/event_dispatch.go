package gateway

import gatewayevents "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"

const (
	TopicMessageReceived = gatewayevents.TopicMessageReceived
	TopicMessageSent     = gatewayevents.TopicMessageSent
	TopicSessionStarted  = gatewayevents.TopicSessionStarted
	TopicSessionEnded    = gatewayevents.TopicSessionEnded
)

type MessageEventPayload = gatewayevents.MessageEventPayload

type EventDispatcher = gatewayevents.EventDispatcher

func NewEventDispatcher(bus gatewayevents.EventBus) *EventDispatcher {
	return gatewayevents.NewEventDispatcher(bus)
}

// MessageEventPayloadFromInbound preserves the provenance and parsed command
// body from a gateway-normalized inbound event.
func MessageEventPayloadFromInbound(ev InboundEvent) MessageEventPayload {
	return MessageEventPayload{
		Platform:         ev.Platform,
		AccountID:        ev.AccountID,
		ChatID:           ev.ChatID,
		ChatName:         ev.ChatName,
		ChatType:         ev.ChatType,
		ParentChatID:     ev.ParentChatID,
		GuildID:          ev.GuildID,
		UserID:           ev.UserID,
		UserName:         ev.UserName,
		ThreadID:         ev.ThreadID,
		MessageID:        ev.MessageID,
		MsgID:            ev.MsgID,
		ReplyToMessageID: "",
		Kind:             ev.Kind.String(),
		Text:             ev.Text,
		Body:             ev.SubmitText(),
	}
}
