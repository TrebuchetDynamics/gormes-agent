package tui

import (
	"strings"
	"testing"
)

func TestHermesChromeRendersGoalBarAboveTodoPanel(t *testing.T) {
	out := RenderHermesChrome(HermesChromeInput{
		Width:        80,
		Conversation: "transcript",
		GoalBar:      "⊙ goal · 3/20 turns · ship parity",
		TodoPanel:    "○ todo item",
		Prompt:       "❯ ",
	})
	goalIndex := strings.Index(out, "⊙ goal")
	todoIndex := strings.Index(out, "○ todo item")
	if goalIndex < 0 || todoIndex < 0 {
		t.Fatalf("chrome missing goal or todo: %q", out)
	}
	if goalIndex > todoIndex {
		t.Fatalf("goal bar should render above todo panel: %q", out)
	}
}
