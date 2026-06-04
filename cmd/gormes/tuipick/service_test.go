package tuipick

import (
	"errors"
	"testing"

	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

func TestShouldFallbackOnlyMatchesRequiresTTY(t *testing.T) {
	if !ShouldFallback(setupwizard.ErrRequiresTTY) {
		t.Fatal("ShouldFallback must match ErrRequiresTTY")
	}
	if !ShouldFallback(errors.Join(errors.New("wrapped"), setupwizard.ErrRequiresTTY)) {
		t.Fatal("ShouldFallback must match joined ErrRequiresTTY")
	}
	if ShouldFallback(errors.New("other")) {
		t.Fatal("ShouldFallback matched unrelated error")
	}
}

func TestWizardChoicesPreserveIDsAndLabels(t *testing.T) {
	got := wizardChoices([]Choice{{ID: "0", Label: "Nous Portal"}, {ID: "1", Label: "OpenAI Codex"}})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "0" || got[0].Label != "Nous Portal" {
		t.Fatalf("choice[0] = %+v", got[0])
	}
	if got[1].ID != "1" || got[1].Label != "OpenAI Codex" {
		t.Fatalf("choice[1] = %+v", got[1])
	}
}
