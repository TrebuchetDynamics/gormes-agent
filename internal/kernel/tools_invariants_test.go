package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TestToolLoop_DoesNotBreakReplaceLatestMailbox proves Phase-1.5's
// replace-latest render-mailbox invariant survives tool execution.
// A 500ms-slow MockTool sits in the middle of a two-round turn while a
// stalled render consumer holds the capacity-1 mailbox. The kernel must
// not deadlock — the mailbox's drain-then-send in emitFrame should keep
// working even when a tool is running for hundreds of milliseconds.
func TestToolLoop_DoesNotBreakReplaceLatestMailbox(t *testing.T) {
	mc := hermes.NewMockClient()

	// Round 1: LLM requests the "slow" tool.
	mc.Script([]hermes.Event{
		{
			Kind: hermes.EventDone, FinishReason: "tool_calls",
			ToolCalls: []hermes.ToolCall{
				{ID: "c1", Name: "slow", Arguments: json.RawMessage(`{}`)},
			},
		},
	}, "sess-stall-tool")
	// Round 2: streaming answer (100 z-tokens + done).
	events := []hermes.Event{}
	for i := 0; i < 100; i++ {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: "z", TokensOut: i + 1})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"})
	mc.Script(events, "sess-stall-tool")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "slow",
		ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return json.RawMessage(`{"done":true}`), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

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

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial = %v, want PhaseIdle", initial.Phase)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "call slow"}); err != nil {
		t.Fatal(err)
	}

	// STALL for 2s across the tool execution. Kernel must not deadlock.
	time.Sleep(2 * time.Second)

	// Peek the current frame — must be the LATEST state, not stale.
	var peeked RenderFrame
	select {
	case peeked = <-k.Render():
	default:
		t.Fatal("no frame available after 2s stall — kernel may have deadlocked during tool execution")
	}

	// Acceptable states: Idle with full 100-z assistant history, or
	// Streaming with draft that prefixes the expected assistant content.
	wantAssistant := strings.Repeat("z", 100)
	ok := false
	if peeked.Phase == PhaseIdle {
		if a := lastAssistantMessage(peeked.History); a != nil && a.Content == wantAssistant {
			ok = true
		}
	}
	if peeked.Phase == PhaseStreaming && peeked.DraftText != "" && strings.HasPrefix(wantAssistant, peeked.DraftText) {
		ok = true
	}
	if !ok {
		t.Errorf("stale peek: phase=%v draftLen=%d historyLen=%d",
			peeked.Phase, len(peeked.DraftText), len(peeked.History))
	}

	// Drain remainder so kernel can exit cleanly on ctx timeout.
	go func() {
		for range k.Render() {
		}
	}()
}
