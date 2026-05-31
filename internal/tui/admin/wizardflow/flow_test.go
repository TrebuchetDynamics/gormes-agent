package wizardflow

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

func TestFlowAdvancesAndStoresTypedAnswers(t *testing.T) {
	flow := New([]wizard.Step{
		wizard.Text("name", "Name"),
		wizard.Pick("provider", "Provider", []wizard.Choice{{ID: "openai", Label: "OpenAI"}}),
		wizard.Confirm("confirm", "Confirm?"),
	})

	step, ok := flow.ActiveStep()
	if !ok || step.ID != "name" || flow.Index() != 0 || flow.Len() != 3 {
		t.Fatalf("initial step = (%q, %v), index=%d len=%d", step.ID, ok, flow.Index(), flow.Len())
	}
	if flow.Finish(wizard.Answer{Text: "agent"}) {
		t.Fatal("first step unexpectedly finished flow")
	}
	if got := flow.Text("name"); got != "agent" {
		t.Fatalf("name answer = %q, want agent", got)
	}

	step, ok = flow.ActiveStep()
	if !ok || step.ID != "provider" {
		t.Fatalf("second step = (%q, %v), want provider", step.ID, ok)
	}
	if flow.Finish(wizard.Answer{Kind: wizard.KindPick, ChoiceID: "openai"}) {
		t.Fatal("second step unexpectedly finished flow")
	}
	if got := flow.Choice("provider"); got != "openai" {
		t.Fatalf("provider answer = %q, want openai", got)
	}

	if !flow.Finish(wizard.Answer{Kind: wizard.KindConfirm, Confirmed: true}) {
		t.Fatal("final step did not finish flow")
	}
	if !flow.Bool("confirm") {
		t.Fatal("confirm answer = false, want true")
	}
	if _, ok := flow.ActiveStep(); ok {
		t.Fatal("active step still available after flow completion")
	}
}

func TestFlowCopiesStepSlice(t *testing.T) {
	steps := []wizard.Step{wizard.Text("one", "One")}
	flow := New(steps)
	steps[0] = wizard.Text("mutated", "Mutated")

	step, ok := flow.ActiveStep()
	if !ok || step.ID != "one" {
		t.Fatalf("active step after caller mutation = (%q, %v), want one", step.ID, ok)
	}
}
