package events

import (
	eventbus "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/bus"
	eventdispatch "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/dispatch"
)

const (
	TopicMessageReceived = eventdispatch.TopicMessageReceived
	TopicMessageSent     = eventdispatch.TopicMessageSent
	TopicSessionStarted  = eventdispatch.TopicSessionStarted
	TopicSessionEnded    = eventdispatch.TopicSessionEnded
)

// MessageEventPayload is the channel-neutral message envelope published on
// gateway message topics. Channel adapters own SDK-specific translation before
// constructing this payload.
type MessageEventPayload = eventdispatch.MessageEventPayload

type EventDispatcher = eventdispatch.EventDispatcher

func NewEventDispatcher(bus eventbus.EventBus) *EventDispatcher {
	return eventdispatch.NewEventDispatcher(bus)
}
