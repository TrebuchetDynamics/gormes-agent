package gateway

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/events"
)

const (
	TopicMessageReceived = "gateway.message.received"
	TopicMessageSent     = "gateway.message.sent"
	TopicSessionStarted  = "gateway.session.started"
	TopicSessionEnded    = "gateway.session.ended"
)

// MessageEventPayload is the channel-neutral message envelope published on
// gateway message topics. Channel adapters own SDK-specific translation before
// constructing this payload.
type MessageEventPayload struct {
	Platform         string `json:"platform"`
	AccountID        string `json:"account_id,omitempty"`
	ChatID           string `json:"chat_id"`
	ChatName         string `json:"chat_name,omitempty"`
	ChatType         string `json:"chat_type,omitempty"`
	UserID           string `json:"user_id,omitempty"`
	UserName         string `json:"user_name,omitempty"`
	ThreadID         string `json:"thread_id,omitempty"`
	MessageID        string `json:"message_id,omitempty"`
	MsgID            string `json:"msg_id,omitempty"`
	ReplyToMessageID string `json:"reply_to_message_id"`
	Kind             string `json:"kind"`
	Text             string `json:"text,omitempty"`
	Body             string `json:"body,omitempty"`
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

type EventDispatcher struct {
	bus events.EventBus
}

func NewEventDispatcher(bus events.EventBus) *EventDispatcher {
	return &EventDispatcher{bus: bus}
}

func (d *EventDispatcher) Available() bool {
	return d != nil && d.bus != nil
}

func (d *EventDispatcher) PublishMessageReceived(source string, traceID string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	evt := events.NewEvent(TopicMessageReceived, source, raw, traceID)
	return d.bus.Publish(TopicMessageReceived, evt)
}

func (d *EventDispatcher) PublishMessageSent(source string, traceID string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	evt := events.NewEvent(TopicMessageSent, source, raw, traceID)
	return d.bus.Publish(TopicMessageSent, evt)
}

func (d *EventDispatcher) PublishSessionStarted(source string, traceID string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	evt := events.NewEvent(TopicSessionStarted, source, raw, traceID)
	return d.bus.Publish(TopicSessionStarted, evt)
}

func (d *EventDispatcher) PublishSessionEnded(source string, traceID string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	evt := events.NewEvent(TopicSessionEnded, source, raw, traceID)
	return d.bus.Publish(TopicSessionEnded, evt)
}

func (d *EventDispatcher) SubscribeMessages(handler events.EventHandler) func() {
	return d.bus.Subscribe(TopicMessageReceived, handler)
}
