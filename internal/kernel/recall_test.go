package kernel

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

// mockRecall implements RecallProvider for kernel-level tests.
type mockRecall struct {
	returnContent string
	delay         time.Duration
	calls         int
	lastInput     RecallParams
}

func (m *mockRecall) GetContext(ctx context.Context, p RecallParams) string {
	m.calls++
	m.lastInput = p
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "" // honor the kernel's deadline cutoff
		}
	}
	return m.returnContent
}

func TestKernel_InjectsMemoryContextWhenRecallNonNil(t *testing.T) {
	rec := &mockRecall{returnContent: "<memory-context>MEMORY BLOCK HERE</memory-context>"}
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-recall-test")

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Recall:    rec,
		ChatKey:   "telegram:42",
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "tell me about Acme"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	req := reqs[0]
	if len(req.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2 (system + user)", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Errorf("Messages[0].Role = %q, want system", req.Messages[0].Role)
	}
	if !strings.Contains(req.Messages[0].Content, "MEMORY BLOCK HERE") {
		t.Errorf("system message doesn't contain mock content: %q", req.Messages[0].Content)
	}
	if req.Messages[1].Role != "user" || req.Messages[1].Content != "tell me about Acme" {
		t.Errorf("Messages[1] = %+v, want user/'tell me about Acme'", req.Messages[1])
	}

	if rec.lastInput.UserMessage != "tell me about Acme" {
		t.Errorf("recall received UserMessage = %q", rec.lastInput.UserMessage)
	}
	if rec.lastInput.ChatKey != "telegram:42" {
		t.Errorf("recall received ChatKey = %q", rec.lastInput.ChatKey)
	}
}

func TestKernel_NoRecallWhenProviderNil(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-no-recall")

	k := New(Config{
		Model: "hermes-agent", Endpoint: "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		// Recall intentionally nil.
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("no requests")
	}
	if len(reqs[0].Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1 (user only, nil Recall)", len(reqs[0].Messages))
	}
}

func TestKernel_RecallTimeoutFallsThrough(t *testing.T) {
	rec := &mockRecall{
		returnContent: "<memory-context>SLOW</memory-context>",
		delay:         500 * time.Millisecond,
	}
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-recall-timeout")

	k := New(Config{
		Model: "hermes-agent", Endpoint: "http://mock",
		Admission:      Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Recall:         rec,
		RecallDeadline: 50 * time.Millisecond,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "slow test"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 3*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("no requests")
	}
	// Timeout path: exactly one message (user). No system injection.
	if len(reqs[0].Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1 (timeout fell through)", len(reqs[0].Messages))
	}
}

func TestKernel_RecallEmptyStringNotInjected(t *testing.T) {
	rec := &mockRecall{returnContent: ""} // empty = nothing to inject
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-empty-recall")

	k := New(Config{
		Model: "hermes-agent", Endpoint: "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Recall:    rec,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "empty recall test"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs[0].Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1 (empty recall)", len(reqs[0].Messages))
	}
	if rec.calls != 1 {
		t.Errorf("recall.calls = %d, want 1 (should still be invoked)", rec.calls)
	}
}
