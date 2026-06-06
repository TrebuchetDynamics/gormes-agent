package eventstream

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit/execution"
)

const (
	TopicToolStart    = "tool.execution.start"
	TopicToolOutput   = "tool.execution.output"
	TopicToolProgress = "tool.execution.progress"
	TopicToolComplete = "tool.execution.complete"
	TopicToolError    = "tool.execution.error"
)

// ToolExecutionPayload is the channel-neutral event payload used by TUI,
// gateway, dashboard, and audit subscribers to follow tool execution.
type ToolExecutionPayload struct {
	AgentID   string            `json:"agent_id,omitempty"`
	ToolName  string            `json:"tool_name"`
	CallID    string            `json:"call_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	EventType string            `json:"event_type"`
	Output    json.RawMessage   `json:"output,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// ToolEventEmitter publishes structured tool execution events on the shared
// event bus.
type ToolEventEmitter struct {
	bus events.EventBus
}

func NewToolEventEmitter(bus events.EventBus) *ToolEventEmitter {
	return &ToolEventEmitter{bus: bus}
}

func (e *ToolEventEmitter) EmitStart(source string, req execution.ToolRequest) error {
	return e.emit(TopicToolStart, source, req, nil, nil)
}

func (e *ToolEventEmitter) EmitOutput(source string, req execution.ToolRequest, output json.RawMessage) error {
	return e.emit(TopicToolOutput, source, req, output, nil)
}

func (e *ToolEventEmitter) EmitProgress(source string, req execution.ToolRequest, output json.RawMessage) error {
	return e.emit(TopicToolProgress, source, req, output, nil)
}

func (e *ToolEventEmitter) EmitComplete(source string, req execution.ToolRequest) error {
	return e.emit(TopicToolComplete, source, req, nil, nil)
}

func (e *ToolEventEmitter) EmitError(source string, req execution.ToolRequest, err error) error {
	if err == nil {
		err = errors.New("tool execution failed")
	}
	return e.emit(TopicToolError, source, req, nil, err)
}

func (e *ToolEventEmitter) emit(topic, source string, req execution.ToolRequest, output json.RawMessage, emitErr error) error {
	if e == nil || e.bus == nil {
		return nil
	}
	callID := toolEventCallID(req.Metadata)
	payload := ToolExecutionPayload{
		AgentID:   req.AgentID,
		ToolName:  req.ToolName,
		CallID:    callID,
		Metadata:  cloneToolEventMetadata(req.Metadata),
		EventType: topic,
		Output:    cloneToolEventRaw(output),
	}
	if emitErr != nil {
		payload.Error = emitErr.Error()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return e.bus.Publish(topic, events.NewEvent(topic, source, raw, toolEventTraceID(req, callID)))
}

// EventingToolExecutor wraps a ToolExecutor and mirrors execution lifecycle
// observations onto the shared event bus.
type EventingToolExecutor struct {
	inner   execution.ToolExecutor
	emitter *ToolEventEmitter
	source  string
}

func NewEventingToolExecutor(inner execution.ToolExecutor, bus events.EventBus, source string) *EventingToolExecutor {
	return &EventingToolExecutor{
		inner:   inner,
		emitter: NewToolEventEmitter(bus),
		source:  source,
	}
}

func (e *EventingToolExecutor) Execute(ctx context.Context, req execution.ToolRequest) (<-chan execution.ToolEvent, error) {
	if e == nil || e.inner == nil {
		err := errors.New("tools: nil eventing tool executor")
		_ = e.emitError(req, err)
		return nil, err
	}
	stream, err := e.inner.Execute(ctx, req)
	if err != nil {
		_ = e.emitError(req, err)
		return nil, err
	}
	if stream == nil {
		err := errors.New("tools: nil tool event stream")
		_ = e.emitError(req, err)
		return nil, err
	}

	out := make(chan execution.ToolEvent, 8)
	go func() {
		defer close(out)
		for event := range stream {
			_ = e.emitFromToolEvent(req, event)
			out <- event
		}
	}()
	return out, nil
}

func (e *EventingToolExecutor) emitFromToolEvent(req execution.ToolRequest, event execution.ToolEvent) error {
	switch event.Type {
	case "started":
		return e.emitter.EmitStart(e.source, req)
	case "output":
		return e.emitter.EmitOutput(e.source, req, event.Output)
	case "completed":
		return e.emitter.EmitComplete(e.source, req)
	case "failed":
		return e.emitError(req, event.Err)
	default:
		return e.emitter.EmitProgress(e.source, req, event.Output)
	}
}

func (e *EventingToolExecutor) emitError(req execution.ToolRequest, err error) error {
	if e == nil || e.emitter == nil {
		return nil
	}
	return e.emitter.EmitError(e.source, req, err)
}

func toolEventCallID(metadata map[string]string) string {
	for _, key := range []string{"call_id", "tool_call_id", "trace_id", "request_id"} {
		if value := metadata[key]; value != "" {
			return value
		}
	}
	return ""
}

func toolEventTraceID(req execution.ToolRequest, callID string) string {
	if callID != "" {
		return callID
	}
	if req.AgentID != "" {
		return req.AgentID + ":" + req.ToolName
	}
	return req.ToolName
}

func cloneToolEventMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func cloneToolEventRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make(json.RawMessage, len(raw))
	copy(out, raw)
	return out
}
