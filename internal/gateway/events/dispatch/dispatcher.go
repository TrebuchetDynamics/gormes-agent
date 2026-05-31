package dispatch

import (
	"encoding/json"

	eventcontract "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/contract"
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
	ParentChatID     string `json:"parent_chat_id,omitempty"`
	GuildID          string `json:"guild_id,omitempty"`
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

type EventDispatcher struct {
	bus eventcontract.EventBus
}

func NewEventDispatcher(bus eventcontract.EventBus) *EventDispatcher {
	return &EventDispatcher{bus: bus}
}

func (d *EventDispatcher) Available() bool {
	return d != nil && d.bus != nil
}

func (d *EventDispatcher) PublishMessageReceived(source string, traceID string, payload interface{}) error {
	return d.publish(TopicMessageReceived, source, traceID, payload)
}

func (d *EventDispatcher) PublishMessageSent(source string, traceID string, payload interface{}) error {
	return d.publish(TopicMessageSent, source, traceID, payload)
}

func (d *EventDispatcher) PublishSessionStarted(source string, traceID string, payload interface{}) error {
	return d.publish(TopicSessionStarted, source, traceID, payload)
}

func (d *EventDispatcher) PublishSessionEnded(source string, traceID string, payload interface{}) error {
	return d.publish(TopicSessionEnded, source, traceID, payload)
}

func (d *EventDispatcher) SubscribeMessages(handler eventcontract.EventHandler) func() {
	return d.bus.Subscribe(TopicMessageReceived, handler)
}

func (d *EventDispatcher) publish(topic string, source string, traceID string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	evt := eventcontract.NewEvent(topic, source, raw, traceID)
	return d.bus.Publish(topic, evt)
}
