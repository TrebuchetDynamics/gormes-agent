package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TestKernel_ToolCallHandshake_Echo is the Tool-Call Handshake.
// Proves Gormes can call its own built-in EchoTool and resume the
// conversation perfectly — the SSE → tool-execution → history-append
// → response-finalisation path works end-to-end with general agent
// skills. External domain tools (scientific simulators, business
// wrappers) inherit this contract by satisfying the same tools.Tool
// interface; Gormes itself ships no domain-specific tools.
func TestKernel_ToolCallHandshake_Echo(t *testing.T) {
	mc := llm.NewMockClient()

	// Round 1: LLM requests the built-in "echo" tool with deterministic args.
	mc.Script([]llm.Event{
		{
			Kind: llm.EventDone, FinishReason: "tool_calls",
			ToolCalls: []llm.ToolCall{
				{
					ID:        "call_echo_1",
					Name:      "echo",
					Arguments: json.RawMessage(`{"text":"GoCo factory online"}`),
				},
			},
		},
	}, "sess-echo")

	// Round 2: LLM's final answer referencing the echoed text.
	finalAnswer := "Tool said: GoCo factory online."
	events := []llm.Event{}
	for _, ch := range finalAnswer {
		events = append(events, llm.Event{Kind: llm.EventToken, Token: string(ch), TokensOut: 1})
	}
	events = append(events, llm.Event{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 50, TokensOut: len(finalAnswer)})
	mc.Script(events, "sess-echo")

	// Register Gormes's built-in EchoTool — no external/domain tools.
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render() // initial idle
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "echo 'GoCo factory online'"}); err != nil {
		t.Fatal(err)
	}

	// Wait for final Idle frame carrying the round-2 assistant message.
	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		if f.Phase != PhaseIdle {
			return false
		}
		a := lastAssistantMessage(f.History)
		return a != nil && a.Content == finalAnswer
	}, 5*time.Second)

	// Sanity checks:
	a := lastAssistantMessage(final.History)
	if a == nil || a.Content != finalAnswer {
		var got string
		if a != nil {
			got = a.Content
		}
		t.Fatalf("final assistant content = %q, want %q", got, finalAnswer)
	}
	if !strings.Contains(a.Content, "GoCo factory online") {
		t.Errorf("final answer doesn't reference the echoed payload: %q", a.Content)
	}
}

func TestKernel_ClearsToolProgressSoulAtTurnStart(t *testing.T) {
	mc := llm.NewMockClient()
	finalAnswer := "No tools needed."
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: finalAnswer, TokensOut: len(finalAnswer)},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 10, TokensOut: len(finalAnswer)},
	}, "sess-clean")

	k := New(Config{
		Model:           "hermes-agent",
		Endpoint:        "http://mock",
		Admission:       Admission{MaxBytes: 200_000, MaxLines: 10_000},
		MaxToolDuration: 5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)
	k.soul = []SoulEntry{{At: time.Now(), Text: "tool: terminal: stale command from previous turn"}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "answer directly"}); err != nil {
		t.Fatal(err)
	}

	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && lastAssistantMessage(f.History) != nil
	}, 5*time.Second)
	for _, event := range final.SoulEvents {
		if strings.Contains(event.Text, "stale command") {
			t.Fatalf("stale tool progress survived into new turn: %#v", final.SoulEvents)
		}
	}
}

func TestKernel_DefaultToolIterationBudgetMatchesHermesBeyondTen(t *testing.T) {
	mc := llm.NewMockClient()
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	for i := 0; i < 11; i++ {
		mc.Script([]llm.Event{{
			Kind:         llm.EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []llm.ToolCall{{
				ID:        "call_echo_" + string(rune('a'+i)),
				Name:      "echo",
				Arguments: json.RawMessage(`{"text":"budget parity"}`),
			}},
		}}, "sess-budget")
	}

	finalAnswer := "Completed after more than ten tool rounds."
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: finalAnswer, TokensOut: len(finalAnswer)},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 50, TokensOut: len(finalAnswer)},
	}, "sess-budget")

	k := New(Config{
		Model:           "hermes-agent",
		Endpoint:        "http://mock",
		Admission:       Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:           reg,
		MaxToolDuration: 5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "exercise default iteration budget"}); err != nil {
		t.Fatal(err)
	}

	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		if f.Phase == PhaseFailed {
			return true
		}
		if f.Phase != PhaseIdle {
			return false
		}
		a := lastAssistantMessage(f.History)
		return a != nil && a.Content == finalAnswer
	}, 5*time.Second)

	if final.Phase == PhaseFailed {
		t.Fatalf("default tool budget failed early with LastError=%q; Hermes default allows 90 iterations", final.LastError)
	}
}

func TestKernel_ToolIterationSummaryDoesNotMixPartialDraft(t *testing.T) {
	mc := llm.NewMockClient()
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	// Round 1 (iteration 1) executes a tool.
	mc.Script([]llm.Event{{
		Kind:         llm.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []llm.ToolCall{{
			ID:        "call_echo_a",
			Name:      "echo",
			Arguments: json.RawMessage(`{"text":"budget parity"}`),
		}},
	}}, "sess-mix-draft")

	// Round 2 (over budget) streams partial assistant text *before* its tool
	// calls. That leftover text must not leak into the final summary.
	leftover := "Let me check one more thing... "
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: leftover, TokensOut: len(leftover)},
		{
			Kind:         llm.EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []llm.ToolCall{{
				ID:        "call_echo_b",
				Name:      "echo",
				Arguments: json.RawMessage(`{"text":"again"}`),
			}},
		},
	}, "sess-mix-draft")

	summary := "Here is the clean final summary."
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: summary, TokensOut: len(summary)},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 50, TokensOut: len(summary)},
	}, "sess-mix-draft")

	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 1,
		MaxToolDuration:   5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "mix draft check"}); err != nil {
		t.Fatal(err)
	}

	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		if f.Phase == PhaseFailed {
			t.Fatalf("budget exhaustion surfaced as failure LastError=%q", f.LastError)
		}
		if f.Phase != PhaseIdle {
			return false
		}
		return lastAssistantMessage(f.History) != nil
	}, 5*time.Second)

	got := lastAssistantMessage(final.History)
	if got == nil {
		t.Fatal("no final assistant message")
	}
	if got.Content != summary {
		t.Fatalf("final assistant content = %q, want only the summary %q (partial draft leaked)", got.Content, summary)
	}
	if strings.Contains(got.Content, leftover) {
		t.Fatalf("final assistant content leaked partial tool-round draft: %q", got.Content)
	}
}

func TestKernel_ToolIterationBudgetRequestsToollessSummary(t *testing.T) {
	mc := llm.NewMockClient()
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	for i := 0; i < 2; i++ {
		mc.Script([]llm.Event{{
			Kind:         llm.EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []llm.ToolCall{{
				ID:        "call_echo_" + string(rune('a'+i)),
				Name:      "echo",
				Arguments: json.RawMessage(`{"text":"budget parity"}`),
			}},
		}}, "sess-budget-summary")
	}

	summary := "I reached the tool budget, so here is the useful summary."
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: summary, TokensOut: len(summary)},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 50, TokensOut: len(summary)},
	}, "sess-budget-summary")

	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 1,
		MaxToolDuration:   5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "exercise summary budget parity"}); err != nil {
		t.Fatal(err)
	}

	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		if f.Phase == PhaseFailed {
			t.Fatalf("budget exhaustion surfaced as failure LastError=%q", f.LastError)
		}
		if f.Phase != PhaseIdle {
			return false
		}
		a := lastAssistantMessage(f.History)
		return a != nil && a.Content == summary
	}, 5*time.Second)

	if final.LastError != "" {
		t.Fatalf("final LastError = %q, want empty after summary fallback", final.LastError)
	}
	reqs := mc.Requests()
	if len(reqs) != 3 {
		t.Fatalf("OpenStream request count = %d, want 3", len(reqs))
	}
	if len(reqs[2].Tools) != 0 {
		t.Fatalf("summary request tools = %d, want no tools", len(reqs[2].Tools))
	}
	last := reqs[2].Messages[len(reqs[2].Messages)-1]
	if last.Role != "user" || !strings.Contains(last.Content, "maximum number of tool-calling iterations") {
		t.Fatalf("summary request last message = %#v, want Hermes-style summary request", last)
	}
}
