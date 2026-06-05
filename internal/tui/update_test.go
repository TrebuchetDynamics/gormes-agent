package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// R3: when a turn is active but nothing else signals progress yet (no tool
// trace, no draft, no error), the live frame shows the reused thinking
// indicator so the user is never left wondering.
func TestStreamFeedback_ThinkingIndicatorWhenActiveAndQuiet(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		History: []llm.Message{
			{Role: "user", Content: "do the thing"},
		},
	}
	got := renderConv(frame, 100, 12)

	if !strings.Contains(got, "🤔") || !strings.Contains(got, "Reasoning") {
		t.Fatalf("active+quiet turn missing reused thinking indicator:\n%s", got)
	}
	if !strings.Contains(got, "do the thing") {
		t.Fatalf("user content dropped:\n%s", got)
	}
}

// The thinking indicator must not appear once a concrete signal exists
// (draft text, tool progress) — that would be redundant noise and could
// disturb transcript ordering. This also protects the fidelity goldens.
func TestStreamFeedback_ThinkingIndicatorSuppressedWhenDraftOrToolPresent(t *testing.T) {
	withDraft := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		History:   []llm.Message{{Role: "user", Content: "q"}},
		DraftText: "partial answer streaming in",
	}
	if g := renderConv(withDraft, 100, 12); strings.Contains(g, "🤔") {
		t.Fatalf("thinking indicator must be suppressed once draft text exists:\n%s", g)
	}

	withTool := kernel.RenderFrame{
		Phase:      kernel.PhaseStreaming,
		History:    []llm.Message{{Role: "user", Content: "q"}},
		SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: "tool: read_file: x.go"}},
	}
	if g := renderConv(withTool, 100, 12); strings.Contains(g, "🤔") {
		t.Fatalf("thinking indicator must be suppressed once tool progress exists:\n%s", g)
	}

	idle := kernel.RenderFrame{}
	if g := renderConv(idle, 80, 8); strings.Contains(g, "🤔") {
		t.Fatalf("idle/empty frame must not show thinking indicator (empty intro instead):\n%s", g)
	}
}

// R3: long tool output collapses to a head plus a "[+N more lines]" summary
// (ccx-go RenderToolOutputInline pattern); short output stays intact so the
// fidelity small-content guarantee is preserved.
func TestStreamFeedback_CollapsibleToolOutput(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 12; i++ {
		b.WriteString("line ")
		b.WriteString(strings.Repeat("x", 3))
		b.WriteByte('\n')
		b.WriteString(itoa(i))
		b.WriteByte('\n')
	}
	long := kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		History: []llm.Message{
			{Role: "user", Content: "read big"},
			{Role: "tool", Name: "read_file", Content: b.String()},
		},
	}
	got := renderConv(long, 100, 40)
	if !strings.Contains(got, "more lines]") {
		t.Fatalf("long tool output not collapsed (missing '[+N more lines]'):\n%s", got)
	}
	if strings.Contains(got, itoa(12)) {
		t.Fatalf("collapsed tool output leaked a tail line (saw %q):\n%s", itoa(12), got)
	}

	short := kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		History: []llm.Message{
			{Role: "user", Content: "read small"},
			{Role: "tool", Name: "read_file", Content: "package tui\n\nfunc View() string"},
		},
	}
	g2 := renderConv(short, 100, 20)
	if strings.Contains(g2, "more lines]") {
		t.Fatalf("short tool output must not be collapsed:\n%s", g2)
	}
	if !strings.Contains(g2, "package tui") || !strings.Contains(g2, "func View() string") {
		t.Fatalf("short tool output truncated unexpectedly:\n%s", g2)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
