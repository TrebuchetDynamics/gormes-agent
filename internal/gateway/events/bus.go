package events

import eventbus "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/bus"

// EventBus decouples publishers from subscribers via topic-based routing.
// Implementations must be safe for concurrent use.
type EventBus = eventbus.EventBus

// InProcessEventBus is an in-memory, topic-based pub/sub bus with no
// external dependencies. Safe for concurrent use.
type InProcessEventBus = eventbus.InProcessEventBus

// NewInProcessEventBus creates a new in-process event bus with the default
// subscriber buffer size.
func NewInProcessEventBus() *InProcessEventBus {
	return eventbus.NewInProcessEventBus()
}
