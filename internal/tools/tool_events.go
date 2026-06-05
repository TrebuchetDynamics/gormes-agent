package tools

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const (
	TopicToolStart    = toolkit.TopicToolStart
	TopicToolOutput   = toolkit.TopicToolOutput
	TopicToolProgress = toolkit.TopicToolProgress
	TopicToolComplete = toolkit.TopicToolComplete
	TopicToolError    = toolkit.TopicToolError
)

type ToolExecutionPayload = toolkit.ToolExecutionPayload
type ToolEventEmitter = toolkit.ToolEventEmitter
type EventingToolExecutor = toolkit.EventingToolExecutor

func NewToolEventEmitter(bus events.EventBus) *ToolEventEmitter {
	return toolkit.NewToolEventEmitter(bus)
}

func NewEventingToolExecutor(inner ToolExecutor, bus events.EventBus, source string) *EventingToolExecutor {
	return toolkit.NewEventingToolExecutor(inner, bus, source)
}
