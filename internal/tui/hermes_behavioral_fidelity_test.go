package tui

// Gormes-owned chat-chrome contract (post-divergence).
//
// History: this file used to pin the chat TUI to Hermes `ui-tui`
// presentation (exact intro strings, ❯/┊/─── glyphs, forbidden role
// labels). Per the methodology-first strategy pivot and progress.json
// Phase 8.D "Gormes-owned chat TUI" rows (R0 ratification), the chat
// presentation is now a Gormes-owned surface that may diverge from Hermes
// `ui-tui` by design — see CLAUDE.md "Core rule".
//
// What these tests now guarantee are the genuinely BEHAVIORAL invariants
// that must survive any presentation restyle (R1 welcome panel, R2 semantic
// styles, R3 streaming feedback):
//   - transcript ordering: user content, then tool reference, then
//     assistant content, then status, then the next input affordance;
//   - tool output is never absorbed into assistant prose;
//   - the empty state shows product identity + a help affordance and is not
//     the legacy placeholder;
//   - the final assistant message renders exactly once with no leaked
//     live progress/control text.
//
// They deliberately no longer pin Hermes-specific glyphs, the exact intro
// wording, or forbid Gormes-owned identity/role labels. Genuine legacy /
// debug-noise chrome is still forbidden.
//
// NOTE: contract-readiness.md still references the old "Hermes behavioral
// fidelity" framing; updating that doc is a follow-up outside R0 write_scope.

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// promptSentinel / statusSentinel are presentation-agnostic markers we feed
// into the chrome so transcript-ordering can be asserted without pinning the
// Gormes-owned prompt/status glyphs (which R2/R3 are free to restyle).
const (
	promptSentinel = "<<PROMPT>>"
	statusSentinel = "<<STATUS_RULE>>"
)

func TestHermesBehavioralFidelity_InkTranscriptOrder(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-ink-order",
		History: []llm.Message{
			{Role: "user", Content: "inspect the renderer"},
		},
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: read_file: internal/tui/view.go"},
		},
		DraftText: "The renderer keeps transcript content above the composer.",
	}

	conversation := renderConv(frame, 100, 12)
	got := RenderHermesChrome(HermesChromeInput{
		Width:        100,
		Conversation: conversation,
		StatusBar:    statusSentinel,
		Prompt:       promptSentinel,
	})

	// Behavioral guarantee: transcript ordering, not specific glyphs.
	assertContainsInOrder(t, got,
		"inspect the renderer",
		"read_file",
		"The renderer keeps transcript content above the composer.",
		statusSentinel,
		promptSentinel,
	)
}

func TestHermesBehavioralFidelity_ToolTrailAndResultBox(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		History: []llm.Message{
			{Role: "user", Content: "read the TUI renderer"},
			{Role: "tool", Name: "read_file", Content: "package tui\n\nfunc View() string"},
		},
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: read_file: internal/tui/view.go"},
		},
		DraftText: "The renderer response should not absorb tool output.",
	}

	got := renderConv(frame, 100, 14)

	// Behavioral guarantee: tool reference precedes assistant prose, and
	// tool output is never merged into the assistant message.
	assertContainsInOrder(t, got,
		"read_file",
		"package tui",
		"The renderer response should not absorb tool output.",
	)
	assertContainsBefore(t, got, "internal/tui/view.go", "The renderer response should not absorb tool output.")
	if strings.Contains(got, "package tui The renderer response") {
		t.Fatalf("tool output was mixed into assistant prose:\n%s", got)
	}
}

func TestHermesBehavioralFidelity_QueuedMessagesAndStickyPrompt(t *testing.T) {
	got := RenderHermesChrome(HermesChromeInput{
		Width:          100,
		Conversation:   "<<TRANSCRIPT>>",
		QueuedMessages: "queued (2)\n  1. follow up after this turn\n  2. summarize the patch",
		StickyPrompt:   "inspect the renderer",
		StatusBar:      statusSentinel,
		Prompt:         promptSentinel,
	})

	// Behavioral guarantee: queued work and the sticky prompt stay between
	// the transcript and the status/input affordance, in that order.
	assertContainsInOrder(t, got,
		"<<TRANSCRIPT>>",
		"queued (2)",
		"inspect the renderer",
		statusSentinel,
		promptSentinel,
	)
}

func TestHermesBehavioralFidelity_NoLegacyPromptToolkitChrome(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "done"},
		},
	}
	got := RenderHermesChrome(HermesChromeInput{
		Width:        100,
		Conversation: renderConv(frame, 100, 8),
		StatusBar:    statusSentinel,
		Prompt:       promptSentinel,
	})

	// Genuine legacy / debug-noise chrome is still forbidden. Gormes-owned
	// identity/role labels (e.g. "⚕ Gormes", "you:") are intentionally NOT
	// forbidden anymore — that divergence is the point of R0.
	for _, forbidden := range []string{
		"Telemetry",
		"Soul Monitor",
		"prompt_toolkit",
		"phase:",
		"⚕ Hermes",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("snapshot leaked legacy/debug chrome %q:\n%s", forbidden, got)
		}
	}
	assertContainsInOrder(t, got, "hello", "done")
}

func TestHermesBehavioralFidelity_EmptyTranscriptIntro(t *testing.T) {
	got := renderConv(kernel.RenderFrame{}, 100, 12)

	// Behavioral guarantee: the empty state shows product identity and a
	// help affordance, and is NOT blank or the legacy placeholder. The exact
	// wording is Gormes-owned and free to change (R1 welcome panel).
	if strings.TrimSpace(got) == "" {
		t.Fatalf("empty transcript rendered nothing; expected product identity + help affordance")
	}
	if !strings.Contains(got, "Gormes") {
		t.Fatalf("empty transcript intro missing product identity %q:\n%s", "Gormes", got)
	}
	if !strings.Contains(got, "/help") {
		t.Fatalf("empty transcript intro missing a help affordance %q:\n%s", "/help", got)
	}
	if strings.Contains(got, "start typing below to begin") {
		t.Fatalf("empty transcript leaked the old legacy placeholder:\n%s", got)
	}
}

func TestHermesBehavioralFidelity_InkTurnSeparators(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "first prompt"},
			{Role: "assistant", Content: "first response"},
			{Role: "user", Content: "second prompt"},
			{Role: "assistant", Content: "second response"},
		},
	}

	got := renderConv(frame, 100, 16)

	// Behavioral guarantee: multiple turns render in order and do not
	// collapse or duplicate. The visual separator style is Gormes-owned.
	assertContainsInOrder(t, got,
		"first prompt",
		"first response",
		"second prompt",
		"second response",
	)
}

func TestHermesBehavioralFidelity_FinalAssistantNotDuplicated(t *testing.T) {
	// Preserved verbatim: this is the strongest behavioral guarantee and
	// must keep its teeth across every presentation restyle (R1-R3).
	final := "I reached the tool budget, so here is the useful summary."
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		DraftText: final,
		History: []llm.Message{
			{Role: "user", Content: "use tools until the budget is exhausted"},
			{Role: "assistant", Content: final},
		},
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: `tool: terminal: grep -R "budget" .`},
		},
	}

	got := renderConv(frame, 100, 12)
	if count := strings.Count(got, final); count != 1 {
		t.Fatalf("renderConv rendered final assistant %d times, want 1:\n%s", count, got)
	}
	for _, forbidden := range []string{"tool iteration limit exceeded", "terminal", "grep -R", "Finalizing"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("renderConv leaked live progress/control text %q after final:\n%s", forbidden, got)
		}
	}
}

func assertContainsInOrder(t *testing.T, got string, markers ...string) {
	t.Helper()
	offset := 0
	for _, marker := range markers {
		idx := strings.Index(got[offset:], marker)
		if idx < 0 {
			t.Fatalf("snapshot missing %q after byte %d:\n%s", marker, offset, got)
		}
		offset += idx + len(marker)
	}
}

func assertContainsBefore(t *testing.T, got, before, after string) {
	t.Helper()
	beforeIdx := strings.Index(got, before)
	if beforeIdx < 0 {
		t.Fatalf("snapshot missing %q:\n%s", before, got)
	}
	afterIdx := strings.Index(got, after)
	if afterIdx < 0 {
		t.Fatalf("snapshot missing %q:\n%s", after, got)
	}
	if beforeIdx >= afterIdx {
		t.Fatalf("%q must appear before %q:\n%s", before, after, got)
	}
}
