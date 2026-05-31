package toolkit

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit/eventstream"
)

const (
	TopicToolStart    = eventstream.TopicToolStart
	TopicToolOutput   = eventstream.TopicToolOutput
	TopicToolProgress = eventstream.TopicToolProgress
	TopicToolComplete = eventstream.TopicToolComplete
	TopicToolError    = eventstream.TopicToolError
)

// ToolExecutionPayload is the channel-neutral event payload used by TUI,
// gateway, dashboard, and audit subscribers to follow tool execution.
type ToolExecutionPayload = eventstream.ToolExecutionPayload

// ToolEventEmitter publishes structured tool execution events on the shared
// event bus.
type ToolEventEmitter = eventstream.ToolEventEmitter

// EventingToolExecutor wraps a ToolExecutor and mirrors execution lifecycle
// observations onto the shared event bus.
type EventingToolExecutor = eventstream.EventingToolExecutor

func NewToolEventEmitter(bus events.EventBus) *ToolEventEmitter {
	return eventstream.NewToolEventEmitter(bus)
}

func NewEventingToolExecutor(inner ToolExecutor, bus events.EventBus, source string) *EventingToolExecutor {
	return eventstream.NewEventingToolExecutor(inner, bus, source)
}
