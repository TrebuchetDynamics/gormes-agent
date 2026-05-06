package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestHermesBehavioralFidelity_InkTranscriptOrder(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-ink-order",
		History: []hermes.Message{
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
		StatusBar:    "<<STATUS_RULE>>",
		Prompt:       "❯ ",
	})

	assertContainsInOrder(t, got,
		"you:",
		"inspect the renderer",
		"read_file",
		HermesChromeAssistantLabel(),
		"The renderer keeps transcript content above the composer.",
		"<<STATUS_RULE>>",
		"❯ ",
	)
}

func TestHermesBehavioralFidelity_ToolTrailAndResultBox(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		History: []hermes.Message{
			{Role: "user", Content: "read the TUI renderer"},
			{Role: "tool", Name: "read_file", Content: "package tui\n\nfunc View() string"},
		},
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: read_file: internal/tui/view.go"},
		},
		DraftText: "The renderer response should not absorb tool output.",
	}

	got := renderConv(frame, 100, 14)

	assertContainsInOrder(t, got,
		"tool result: read_file",
		"package tui",
		HermesChromeAssistantLabel(),
		"The renderer response should not absorb tool output.",
	)
	assertContainsBefore(t, got, "internal/tui/view.go", HermesChromeAssistantLabel())
	if strings.Contains(got, "package tui The renderer response") {
		t.Fatalf("tool output was mixed into assistant prose:\n%s", got)
	}
}

func TestHermesBehavioralFidelity_QueuedMessagesAndStickyPrompt(t *testing.T) {
	got := RenderHermesChrome(HermesChromeInput{
		Width:          100,
		Conversation:   "<<TRANSCRIPT>>",
		QueuedMessages: "queued (2)\n  1. follow up after this turn\n  2. summarize the patch",
		StickyPrompt:   "↳ inspect the renderer",
		StatusBar:      "<<STATUS_RULE>>",
		Prompt:         "❯ ",
	})

	assertContainsInOrder(t, got,
		"<<TRANSCRIPT>>",
		"queued (2)",
		"↳ inspect the renderer",
		"<<STATUS_RULE>>",
		"❯ ",
	)
}

func TestHermesBehavioralFidelity_NoLegacyPromptToolkitChrome(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "done"},
		},
	}
	got := RenderHermesChrome(HermesChromeInput{
		Width:        100,
		Conversation: renderConv(frame, 100, 8),
		StatusBar:    "<<STATUS_RULE>>",
		Prompt:       "❯ ",
	})

	for _, forbidden := range []string{
		"Telemetry",
		"Soul Monitor",
		"prompt_toolkit",
		"phase:",
		"⚕ Hermes",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("snapshot leaked legacy chrome %q:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, HermesChromeAssistantLabel()) {
		t.Fatalf("snapshot missing Gormes-owned assistant label:\n%s", got)
	}
}

func TestHermesBehavioralFidelity_FinalAssistantNotDuplicated(t *testing.T) {
	final := "I reached the tool budget, so here is the useful summary."
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		DraftText: final,
		History: []hermes.Message{
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
