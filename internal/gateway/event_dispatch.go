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

type EventDispatcher struct {
	bus events.EventBus
}

func NewEventDispatcher(bus events.EventBus) *EventDispatcher {
	return &EventDispatcher{bus: bus}
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
