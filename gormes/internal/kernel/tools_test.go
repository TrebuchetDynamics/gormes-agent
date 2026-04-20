package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// TestKernel_ToolCallHandshake_Echo is the Tool-Call Handshake.
// Proves Gormes can call its own built-in EchoTool and resume the
// conversation perfectly — the SSE → tool-execution → history-append
// → response-finalisation path works end-to-end with general agent
// skills. External domain tools (scientific simulators, business
// wrappers) inherit this contract by satisfying the same tools.Tool
// interface; Gormes itself ships no domain-specific tools.
func TestKernel_ToolCallHandshake_Echo(t *testing.T) {
	mc := hermes.NewMockClient()

	// Round 1: LLM requests the built-in "echo" tool with deterministic args.
	mc.Script([]hermes.Event{
		{
			Kind: hermes.EventDone, FinishReason: "tool_calls",
			ToolCalls: []hermes.ToolCall{
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
	events := []hermes.Event{}
	for _, ch := range finalAnswer {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: string(ch), TokensOut: 1})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 50, TokensOut: len(finalAnswer)})
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
