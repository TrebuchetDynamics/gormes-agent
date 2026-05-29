package gateway

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestIterationLimitFinalization_GatewaySendsOneSummaryFinal(t *testing.T) {
	ch := newFakeChannel("discord")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("discord", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer cancelCoalescer(coCancel)

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: `tool: terminal: grep -R "budget" .`},
		},
	}, &co, &coCancel)
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:      kernel.PhaseFinalizing,
		StatusText: "iteration budget exhausted (1/1); requesting summary",
	}, &co, &coCancel)

	final := "I reached the tool budget, so here is the useful summary."
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "use tools until the budget is exhausted"},
			{Role: "assistant", Content: final},
		},
	}, &co, &coCancel)

	sent := ch.sentSnapshot()
	if got := countSentTextsContaining(sent, final); got != 1 {
		t.Fatalf("final summary sends = %d, want 1; sent=%#v edits=%#v", got, sent, ch.editsSnapshot())
	}
	for _, msg := range sent {
		if strings.Contains(msg.Text, "tool iteration limit exceeded") {
			t.Fatalf("gateway leaked raw loop-control text in %#v", msg)
		}
	}
	if strings.Contains(joinSentTexts(sent), "terminal") && strings.Contains(final, "terminal") {
		t.Fatalf("test fixture final unexpectedly contains tool progress")
	}
	if strings.Contains(finalMessageText(sent, final), "terminal") {
		t.Fatalf("final answer contains tool progress; sent=%#v", sent)
	}
	if m.hasActiveTurn() {
		t.Fatalf("final iteration summary frame must clear the active turn")
	}
}

func TestIterationLimitFinalization_ToolProgressClearedAfterFinal(t *testing.T) {
	ch := newFakeChannel("discord")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("discord", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer cancelCoalescer(coCancel)

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: `tool: read_file: internal/kernel/kernel.go`},
		},
	}, &co, &coCancel)
	if got := ch.sentSnapshot(); len(got) != 1 || !strings.Contains(got[0].Text, "read_file") {
		t.Fatalf("tool progress send = %#v, want one read_file progress message", got)
	}

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "assistant", Content: "Final summary after tool budget."},
		},
	}, &co, &coCancel)

	m.toolProgressMu.Lock()
	msgID := m.toolProgressMsgID
	text := m.toolProgressText
	m.toolProgressMu.Unlock()
	if msgID != "" || text != "" {
		t.Fatalf("tool progress state = (%q,%q), want cleared after final", msgID, text)
	}
	if got := countSentTextsContaining(ch.sentSnapshot(), "Final summary after tool budget."); got != 1 {
		t.Fatalf("final summary sends = %d, want 1; sent=%#v", got, ch.sentSnapshot())
	}
}

func TestIterationLimitFinalization_SummaryFailureIsBoundedDegradedText(t *testing.T) {
	ch := newFakeChannel("discord")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("discord", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer cancelCoalescer(coCancel)

	degraded := "I reached the maximum iterations (1) but couldn't summarize. Error: provider unavailable"
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "assistant", Content: degraded},
		},
	}, &co, &coCancel)

	sent := ch.sentSnapshot()
	if got := countSentTextsContaining(sent, "couldn't summarize"); got != 1 {
		t.Fatalf("degraded final sends = %d, want 1; sent=%#v", got, sent)
	}
	for _, forbidden := range []string{"tool iteration limit exceeded", "panic:", "<tool_call", "</tool_call>"} {
		if strings.Contains(joinSentTexts(sent), forbidden) {
			t.Fatalf("degraded final leaked %q in sent=%#v", forbidden, sent)
		}
	}
}

func cancelCoalescer(cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
}

func countSentTextsContaining(sent []fakeSent, needle string) int {
	count := 0
	for _, msg := range sent {
		if strings.Contains(msg.Text, needle) {
			count++
		}
	}
	return count
}

func finalMessageText(sent []fakeSent, needle string) string {
	for _, msg := range sent {
		if strings.Contains(msg.Text, needle) {
			return msg.Text
		}
	}
	return ""
}

func joinSentTexts(sent []fakeSent) string {
	var b strings.Builder
	for _, msg := range sent {
		b.WriteString(msg.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
