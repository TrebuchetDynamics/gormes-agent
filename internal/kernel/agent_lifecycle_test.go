package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestAgentLifecycleHook_StartEndFiresOnSimpleTurn(t *testing.T) {
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "hello"},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 10, TokensOut: 5},
	}, "ses-01")

	var mu sync.Mutex
	var events []AgentLifecycleEvent
	hook := func(ctx context.Context, ev AgentLifecycleEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	}

	k := New(Config{
		Model:              "hermes-agent",
		Endpoint:           "http://mock",
		Admission:          Admission{MaxBytes: 200_000, MaxLines: 10_000},
		AgentLifecycleHook: hook,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle
	}, 500*time.Millisecond)

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi", SessionID: "ses-01"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.Seq > 1
	}, 500*time.Millisecond)

	events = waitForAgentLifecycleEvents(t, &mu, &events, 2)
	if len(events) != 2 {
		t.Fatalf("got %d lifecycle events, want 2 (start + end)", len(events))
	}
	if events[0].Point != AgentLifecycleStart {
		t.Errorf("event[0].Point = %q, want agent:start", events[0].Point)
	}
	if events[0].SessionID != "ses-01" {
		t.Errorf("event[0].SessionID = %q, want ses-01", events[0].SessionID)
	}
	if events[0].Iteration != 0 {
		t.Errorf("event[0].Iteration = %d, want 0 for start", events[0].Iteration)
	}
	if events[1].Point != AgentLifecycleEnd {
		t.Errorf("event[1].Point = %q, want agent:end", events[1].Point)
	}
	if events[1].SessionID != "ses-01" {
		t.Errorf("event[1].SessionID = %q, want ses-01", events[1].SessionID)
	}
	if events[1].Err != nil {
		t.Errorf("event[1].Err = %v, want nil for successful turn", events[1].Err)
	}
}

func TestAgentLifecycleHook_StepFiresOnToolTurn(t *testing.T) {
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{
		{Kind: llm.EventDone, FinishReason: "tool_calls", ToolCalls: []llm.ToolCall{
			{Name: "echo", ID: "t1", Arguments: json.RawMessage(`{"msg":"hi"}`)},
		}},
	}, "ses-02")
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "done"},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 20, TokensOut: 8},
	}, "ses-02")

	var mu sync.Mutex
	var events []AgentLifecycleEvent
	hook := func(ctx context.Context, ev AgentLifecycleEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	}

	k := New(Config{
		Model:              "hermes-agent",
		Endpoint:           "http://mock",
		Admission:          Admission{MaxBytes: 200_000, MaxLines: 10_000},
		MaxToolIterations:  10,
		AgentLifecycleHook: hook,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle
	}, 500*time.Millisecond)

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "echo hi", SessionID: "ses-02"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.Seq > 1
	}, 500*time.Millisecond)

	events = waitForAgentLifecycleEvents(t, &mu, &events, 3)
	if len(events) != 3 {
		t.Fatalf("got %d lifecycle events, want 3 (start + step + end)", len(events))
	}
	if events[0].Point != AgentLifecycleStart {
		t.Errorf("event[0].Point = %q, want agent:start", events[0].Point)
	}
	if events[1].Point != AgentLifecycleStep {
		t.Errorf("event[1].Point = %q, want agent:step", events[1].Point)
	}
	if events[1].Iteration != 1 {
		t.Errorf("event[1].Iteration = %d, want 1", events[1].Iteration)
	}
	if len(events[1].ToolNames) != 1 || events[1].ToolNames[0] != "echo" {
		t.Errorf("event[1].ToolNames = %v, want [echo]", events[1].ToolNames)
	}
	if events[2].Point != AgentLifecycleEnd {
		t.Errorf("event[2].Point = %q, want agent:end", events[2].Point)
	}
}

func TestAgentLifecycleHook_EndFiresOnStreamError(t *testing.T) {
	errClient := &streamErrorClient{err: errors.New("Forbidden: provider returned HTML error body")}

	var mu sync.Mutex
	var events []AgentLifecycleEvent
	hook := func(ctx context.Context, ev AgentLifecycleEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	}

	k := New(Config{
		Model:              "hermes-agent",
		Endpoint:           "http://mock",
		Admission:          Admission{MaxBytes: 200_000, MaxLines: 10_000},
		AgentLifecycleHook: hook,
	}, errClient, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle
	}, 500*time.Millisecond)

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "fail", SessionID: "ses-err"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseFailed || (f.Phase == PhaseIdle && f.Seq > 1)
	}, 2*time.Second)

	events = waitForAgentLifecycleEvents(t, &mu, &events, 2)
	foundEnd := false
	for _, ev := range events {
		if ev.Point == AgentLifecycleEnd {
			foundEnd = true
			if ev.Err == nil {
				t.Error("agent:end Err = nil, want non-nil for failed turn")
			}
			break
		}
	}
	if !foundEnd {
		t.Fatal("agent:end did not fire for failed turn")
	}
}

func waitForAgentLifecycleEvents(t *testing.T, mu *sync.Mutex, events *[]AgentLifecycleEvent, want int) []AgentLifecycleEvent {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		mu.Lock()
		if len(*events) >= want {
			got := append([]AgentLifecycleEvent(nil), (*events)...)
			mu.Unlock()
			return got
		}
		mu.Unlock()
		select {
		case <-ticker.C:
		case <-deadline:
			mu.Lock()
			got := append([]AgentLifecycleEvent(nil), (*events)...)
			mu.Unlock()
			t.Fatalf("got %d lifecycle events, want at least %d: %+v", len(got), want, got)
		}
	}
}

type streamErrorClient struct {
	err error
}

func (c *streamErrorClient) ProviderStatus() llm.ProviderStatus {
	return llm.ProviderStatus{Provider: "mock", Runtime: "test"}
}

func (c *streamErrorClient) Health(ctx context.Context) error { return nil }

func (c *streamErrorClient) OpenStream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	return nil, c.err
}

func (c *streamErrorClient) OpenRunEvents(ctx context.Context, sessionID string) (llm.RunEventStream, error) {
	return nil, llm.ErrRunEventsNotSupported
}

func TestAgentLifecycleHook_NilHookNoPanic(t *testing.T) {
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 5, TokensOut: 3},
	}, "ses-nil")

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle
	}, 500*time.Millisecond)

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "ok"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.Seq > 1
	}, 500*time.Millisecond)
}
