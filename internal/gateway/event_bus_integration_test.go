package gateway_test

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestEventBusIntegration_TelegramTurnToolReplyFlow(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	topics := []string{
		gateway.TopicMessageReceived,
		hermes.TopicTurnStart,
		hermes.TopicTurnAction,
		tools.TopicToolStart,
		tools.TopicToolOutput,
		tools.TopicToolComplete,
		hermes.TopicTurnObserve,
		hermes.TopicTurnComplete,
		gateway.TopicMessageSent,
	}
	gatewayView := newEventFlowRecorder(bus, topics)
	tuiView := newEventFlowRecorder(bus, topics)

	dispatcher := gateway.NewEventDispatcher(bus)
	adapter := telegram.NewBusAdapter(dispatcher)
	turns := hermes.NewTurnEventEmitter(bus)
	executor := tools.NewEventingToolExecutor(integrationToolExecutor{}, bus, "agent")

	traceID := "trace-tg-42-99"
	if err := adapter.PublishInboundMessage(traceID, gateway.InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		ChatType:  "private",
		UserID:    "7",
		UserName:  "Ada",
		ThreadID:  "1",
		MsgID:     "99",
		MessageID: "99",
		Kind:      gateway.EventSubmit,
		Text:      "show task status",
	}); err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}
	if err := turns.EmitStart("agent", traceID); err != nil {
		t.Fatalf("EmitStart: %v", err)
	}
	if err := turns.EmitAction("agent", traceID, "kanban_show", json.RawMessage(`{"task_id":"task-1"}`)); err != nil {
		t.Fatalf("EmitAction: %v", err)
	}

	stream, err := executor.Execute(context.Background(), tools.ToolRequest{
		AgentID:  "session-42",
		ToolName: "kanban_show",
		Input:    json.RawMessage(`{"task_id":"task-1"}`),
		Metadata: map[string]string{
			"call_id":    traceID,
			"session_id": "session-42",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for range stream {
	}

	toolResult := json.RawMessage(`{"task_id":"task-1","status":"ready"}`)
	if err := turns.EmitObserve("agent", traceID, toolResult); err != nil {
		t.Fatalf("EmitObserve: %v", err)
	}
	if err := turns.EmitComplete("agent", traceID); err != nil {
		t.Fatalf("EmitComplete: %v", err)
	}
	if err := dispatcher.PublishMessageSent("telegram", traceID, gateway.MessageEventPayload{
		Platform:  "telegram",
		ChatID:    "42",
		ChatType:  "private",
		UserID:    "7",
		ThreadID:  "1",
		MessageID: "reply-100",
		Kind:      "reply",
		Text:      "task-1 is ready",
		Body:      "task-1 is ready",
	}); err != nil {
		t.Fatalf("PublishMessageSent: %v", err)
	}

	gatewayEvents := gatewayView.waitForCount(t, len(topics))
	tuiEvents := tuiView.waitForCount(t, len(topics))

	wantTopics := []string{
		gateway.TopicMessageReceived,
		hermes.TopicTurnStart,
		hermes.TopicTurnAction,
		tools.TopicToolStart,
		tools.TopicToolOutput,
		tools.TopicToolComplete,
		hermes.TopicTurnObserve,
		hermes.TopicTurnComplete,
		gateway.TopicMessageSent,
	}
	assertEventTopics(t, "gateway subscriber", gatewayEvents, wantTopics)
	assertEventTopics(t, "tui subscriber", tuiEvents, wantTopics)

	assertTelegramInboundPayload(t, gatewayEvents[0])
	assertToolPayload(t, eventByTopic(t, gatewayEvents, tools.TopicToolOutput), traceID)
	assertTelegramReplyPayload(t, gatewayEvents[len(gatewayEvents)-1])
}

type integrationToolExecutor struct{}

func (integrationToolExecutor) Execute(context.Context, tools.ToolRequest) (<-chan tools.ToolEvent, error) {
	ch := make(chan tools.ToolEvent, 3)
	go func() {
		defer close(ch)
		ch <- tools.ToolEvent{Type: "started"}
		ch <- tools.ToolEvent{Type: "output", Output: json.RawMessage(`{"task_id":"task-1","status":"ready"}`)}
		ch <- tools.ToolEvent{Type: "completed"}
	}()
	return ch, nil
}

type eventFlowRecorder struct {
	mu     sync.Mutex
	events []events.Event
}

func newEventFlowRecorder(bus events.EventBus, topics []string) *eventFlowRecorder {
	rec := &eventFlowRecorder{}
	for _, topic := range topics {
		bus.Subscribe(topic, rec.record)
	}
	return rec
}

func (r *eventFlowRecorder) record(event events.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *eventFlowRecorder) waitForCount(t *testing.T, want int) []events.Event {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		r.mu.Lock()
		got := append([]events.Event(nil), r.events...)
		r.mu.Unlock()
		if len(got) >= want {
			sort.SliceStable(got, func(i, j int) bool {
				return got[i].Timestamp.Before(got[j].Timestamp)
			})
			return got[:want]
		}
		if time.Now().After(deadline) {
			t.Fatalf("captured %d events, want %d: %#v", len(got), want, got)
		}
		time.Sleep(time.Millisecond)
	}
}

func assertEventTopics(t *testing.T, name string, got []events.Event, want []string) {
	t.Helper()
	gotTopics := make([]string, len(got))
	for i, event := range got {
		gotTopics[i] = event.Type
	}
	if !reflect.DeepEqual(gotTopics, want) {
		t.Fatalf("%s topics = %v, want %v", name, gotTopics, want)
	}
}

func eventByTopic(t *testing.T, got []events.Event, topic string) events.Event {
	t.Helper()
	for _, event := range got {
		if event.Type == topic {
			return event
		}
	}
	t.Fatalf("missing event topic %q in %#v", topic, got)
	return events.Event{}
}

func assertTelegramInboundPayload(t *testing.T, event events.Event) {
	t.Helper()
	var payload gateway.MessageEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode inbound payload: %v", err)
	}
	if event.Type != gateway.TopicMessageReceived || event.Source != "telegram" || event.TraceID != "trace-tg-42-99" {
		t.Fatalf("inbound provenance = type:%q source:%q trace:%q", event.Type, event.Source, event.TraceID)
	}
	if payload.Platform != "telegram" || payload.ChatID != "42" || payload.UserID != "7" || payload.MessageID != "99" {
		t.Fatalf("inbound payload provenance = %+v, want telegram chat/user/message", payload)
	}
	if payload.Kind != "submit" || payload.Body != "show task status" {
		t.Fatalf("inbound kind/body = %q/%q, want submit/show task status", payload.Kind, payload.Body)
	}
}

func assertToolPayload(t *testing.T, event events.Event, traceID string) {
	t.Helper()
	var payload tools.ToolExecutionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode tool payload: %v", err)
	}
	if event.TraceID != traceID {
		t.Fatalf("tool trace = %q, want %q", event.TraceID, traceID)
	}
	if payload.ToolName != "kanban_show" || payload.AgentID != "session-42" || payload.CallID != traceID {
		t.Fatalf("tool payload = %+v, want kanban_show/session-42/%s", payload, traceID)
	}
	if string(payload.Output) != `{"task_id":"task-1","status":"ready"}` {
		t.Fatalf("tool output = %s, want task ready JSON", payload.Output)
	}
}

func assertTelegramReplyPayload(t *testing.T, event events.Event) {
	t.Helper()
	var payload gateway.MessageEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode reply payload: %v", err)
	}
	if event.Type != gateway.TopicMessageSent || event.Source != "telegram" || event.TraceID != "trace-tg-42-99" {
		t.Fatalf("reply provenance = type:%q source:%q trace:%q", event.Type, event.Source, event.TraceID)
	}
	if payload.Platform != "telegram" || payload.ChatID != "42" || payload.Text != "task-1 is ready" {
		t.Fatalf("reply payload = %+v, want Telegram delivery to chat 42", payload)
	}
}
