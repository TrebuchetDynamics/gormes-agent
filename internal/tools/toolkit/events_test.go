package toolkit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/builtins"
)

func TestToolEvents_WrappedExecutorPublishesLifecycle(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()
	eventCh := subscribeToolEventTopics(bus)

	reg := NewRegistry()
	if err := reg.Register(&builtins.EchoTool{}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	exec := NewEventingToolExecutor(NewInProcessToolExecutor(reg), bus, "gateway")

	stream, err := exec.Execute(context.Background(), ToolRequest{
		AgentID:  "agent-main",
		ToolName: "echo",
		Input:    json.RawMessage(`{"text":"hi"}`),
		Metadata: map[string]string{
			"call_id":    "call-123",
			"session_id": "session-1",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	drainToolEvents(stream)

	got := readToolBusEvents(t, eventCh, TopicToolStart, TopicToolOutput, TopicToolComplete)
	if len(got) != 3 {
		t.Fatalf("bus events = %d, want 3", len(got))
	}
	byType := toolEventsByType(got)
	assertToolEventTimestampOrder(t, byType, TopicToolStart, TopicToolOutput, TopicToolComplete)

	assertToolBusEvent(t, byType[TopicToolStart], TopicToolStart, "gateway", "call-123", "agent-main", "echo", "")
	assertToolBusEvent(t, byType[TopicToolOutput], TopicToolOutput, "gateway", "call-123", "agent-main", "echo", "")
	assertToolBusEvent(t, byType[TopicToolComplete], TopicToolComplete, "gateway", "call-123", "agent-main", "echo", "")

	var payload ToolExecutionPayload
	if err := json.Unmarshal(byType[TopicToolOutput].Payload, &payload); err != nil {
		t.Fatalf("output payload: %v", err)
	}
	if !strings.Contains(string(payload.Output), `"hi"`) {
		t.Fatalf("output payload = %s, want echo output", payload.Output)
	}
	if payload.Metadata["session_id"] != "session-1" {
		t.Fatalf("metadata session_id = %q, want session-1", payload.Metadata["session_id"])
	}
}

func TestToolEvents_WrappedExecutorPublishesErrors(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()
	eventCh := subscribeToolEventTopics(bus)

	reg := NewRegistry()
	if err := reg.Register(&builtins.EchoTool{}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	exec := NewEventingToolExecutor(NewInProcessToolExecutor(reg), bus, "gateway")

	_, err := exec.Execute(context.Background(), ToolRequest{
		AgentID:  "agent-main",
		ToolName: "missing",
		Metadata: map[string]string{"tool_call_id": "call-missing"},
	})
	if !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("Execute missing: got %v, want ErrUnknownTool", err)
	}

	got := readToolBusEvents(t, eventCh, TopicToolError)
	byType := toolEventsByType(got)
	assertToolBusEvent(t, byType[TopicToolError], TopicToolError, "gateway", "call-missing", "agent-main", "missing", "unknown tool")

	stream, err := exec.Execute(context.Background(), ToolRequest{
		AgentID:  "agent-main",
		ToolName: "echo",
		Input:    json.RawMessage(`{}`),
		Metadata: map[string]string{"call_id": "call-failed"},
	})
	if err != nil {
		t.Fatalf("Execute echo failure: %v", err)
	}
	drainToolEvents(stream)

	got = readToolBusEvents(t, eventCh, TopicToolStart, TopicToolError)
	byType = toolEventsByType(got)
	assertToolEventTimestampOrder(t, byType, TopicToolStart, TopicToolError)
	assertToolBusEvent(t, byType[TopicToolStart], TopicToolStart, "gateway", "call-failed", "agent-main", "echo", "")
	assertToolBusEvent(t, byType[TopicToolError], TopicToolError, "gateway", "call-failed", "agent-main", "echo", "text")
}

func subscribeToolEventTopics(bus events.EventBus) <-chan events.Event {
	ch := make(chan events.Event, 16)
	for _, topic := range []string{
		TopicToolStart,
		TopicToolOutput,
		TopicToolProgress,
		TopicToolComplete,
		TopicToolError,
	} {
		bus.Subscribe(topic, func(e events.Event) {
			ch <- e
		})
	}
	return ch
}

func drainToolEvents(stream <-chan ToolEvent) {
	for range stream {
	}
}

func readToolBusEvents(t *testing.T, ch <-chan events.Event, wantTypes ...string) []events.Event {
	t.Helper()
	got := make([]events.Event, 0, len(wantTypes))
	deadline := time.After(2 * time.Second)
	for len(got) < len(wantTypes) {
		select {
		case event := <-ch:
			got = append(got, event)
		case <-deadline:
			t.Fatalf("timed out waiting for bus events; got %v, want %v", eventTypes(got), wantTypes)
		}
	}
	if types := eventTypes(got); !sameStringSet(types, wantTypes) {
		t.Fatalf("event types = %v, want set %v", types, wantTypes)
	}
	return got
}

func assertToolBusEvent(t *testing.T, event events.Event, typ, source, callID, agentID, toolName, errContains string) {
	t.Helper()
	if event.Type != typ {
		t.Fatalf("event type = %q, want %q", event.Type, typ)
	}
	if event.Source != source {
		t.Fatalf("%s source = %q, want %q", typ, event.Source, source)
	}
	if event.TraceID != callID {
		t.Fatalf("%s trace_id = %q, want %q", typ, event.TraceID, callID)
	}
	var payload ToolExecutionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("%s payload: %v", typ, err)
	}
	if payload.CallID != callID {
		t.Fatalf("%s call_id = %q, want %q", typ, payload.CallID, callID)
	}
	if payload.AgentID != agentID {
		t.Fatalf("%s agent_id = %q, want %q", typ, payload.AgentID, agentID)
	}
	if payload.ToolName != toolName {
		t.Fatalf("%s tool_name = %q, want %q", typ, payload.ToolName, toolName)
	}
	if errContains != "" && !strings.Contains(payload.Error, errContains) {
		t.Fatalf("%s error = %q, want contains %q", typ, payload.Error, errContains)
	}
}

func eventTypes(events []events.Event) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.Type)
	}
	return out
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[string]int, len(want))
	for _, value := range got {
		counts[value]++
	}
	for _, value := range want {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

func toolEventsByType(got []events.Event) map[string]events.Event {
	out := make(map[string]events.Event, len(got))
	for _, event := range got {
		out[event.Type] = event
	}
	return out
}

func assertToolEventTimestampOrder(t *testing.T, byType map[string]events.Event, orderedTypes ...string) {
	t.Helper()
	for i := 1; i < len(orderedTypes); i++ {
		prev, ok := byType[orderedTypes[i-1]]
		if !ok {
			t.Fatalf("missing event %q", orderedTypes[i-1])
		}
		next, ok := byType[orderedTypes[i]]
		if !ok {
			t.Fatalf("missing event %q", orderedTypes[i])
		}
		if next.Timestamp.Before(prev.Timestamp) {
			t.Fatalf("timestamp order %s=%s before %s=%s", next.Type, next.Timestamp, prev.Type, prev.Timestamp)
		}
	}
}
