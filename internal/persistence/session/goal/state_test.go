package goal

import (
	"strings"
	"testing"
)

func TestNormalizeStateAndActive(t *testing.T) {
	state := NormalizeState(State{
		Goal:         "  ship it  ",
		Status:       Status(" ACTIVE "),
		TurnsUsed:    -1,
		MaxTurns:     0,
		LastVerdict:  " DONE ",
		LastReason:   "  because  ",
		PausedReason: "  waiting  ",
	})
	if state.Goal != "ship it" || state.Status != StatusActive || state.TurnsUsed != 0 || state.MaxTurns != DefaultMaxTurns {
		t.Fatalf("NormalizeState() = %+v, want trimmed active defaults", state)
	}
	if state.LastVerdict != "done" || state.LastReason != "because" || state.PausedReason != "waiting" {
		t.Fatalf("NormalizeState evidence fields = %+v, want normalized evidence", state)
	}
	if !IsActive(&state) {
		t.Fatal("IsActive(normalized active goal) = false, want true")
	}
}

func TestContinuationPromptIncludesSubgoals(t *testing.T) {
	got := ContinuationPrompt(" finish gormes ", []string{"tests", "docs"})
	for _, want := range []string{"Goal: finish gormes", "1. tests", "2. docs", "Continue working toward this goal."} {
		if !strings.Contains(got, want) {
			t.Fatalf("ContinuationPrompt() missing %q in %q", want, got)
		}
	}
}
