package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

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
