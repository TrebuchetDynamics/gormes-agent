package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestAgentLifecycleHook_StartEndFiresOnSimpleTurn(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "hello"},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 10, TokensOut: 5},
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

	mu.Lock()
	defer mu.Unlock()
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
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "tool_calls", ToolCalls: []hermes.ToolCall{
			{Name: "echo", ID: "t1", Arguments: json.RawMessage(`{"msg":"hi"}`)},
		}},
	}, "ses-02")
	mc.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "done"},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 20, TokensOut: 8},
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

	mu.Lock()
	defer mu.Unlock()
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

	mu.Lock()
	defer mu.Unlock()
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

type streamErrorClient struct {
	err error
}

func (c *streamErrorClient) ProviderStatus() hermes.ProviderStatus {
	return hermes.ProviderStatus{Provider: "mock", Runtime: "test"}
}

func (c *streamErrorClient) Health(ctx context.Context) error { return nil }

func (c *streamErrorClient) OpenStream(ctx context.Context, req hermes.ChatRequest) (hermes.Stream, error) {
	return nil, c.err
}

func (c *streamErrorClient) OpenRunEvents(ctx context.Context, sessionID string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func TestAgentLifecycleHook_NilHookNoPanic(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 5, TokensOut: 3},
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
