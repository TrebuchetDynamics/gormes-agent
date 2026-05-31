package wizardflow

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	tea "github.com/charmbracelet/bubbletea"
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

func TestNewInputConfiguresSharedWizardTextEntry(t *testing.T) {
	input := NewInput(InputOptions{Width: 12, Value: "default", Password: true})
	if got := input.Prompt; got != "> " {
		t.Fatalf("prompt = %q, want default", got)
	}
	if got := input.Width; got != 12 {
		t.Fatalf("width = %d, want 12", got)
	}
	if got := input.Value(); got != "default" {
		t.Fatalf("value = %q, want default", got)
	}
	if got := input.View(); got == "" || got == "default" {
		t.Fatalf("password input view exposed value or was empty: %q", got)
	}
}

func TestUpdateInputAppliesKeyWithoutCallerCommandPlumbing(t *testing.T) {
	input := NewInput(InputOptions{})
	input = UpdateInput(input, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o', 'k'}})
	if got := input.Value(); got != "ok" {
		t.Fatalf("input value = %q, want ok", got)
	}
}

func TestPickHelpersClampMovementAndBuildAnswer(t *testing.T) {
	step := wizard.Pick("provider", "Provider", []wizard.Choice{
		{ID: "openai", Label: "OpenAI"},
		{ID: "anthropic", Label: "Anthropic"},
	})
	if got := MovePickCursor(0, step, -1); got != 0 {
		t.Fatalf("move above first = %d, want 0", got)
	}
	cursor := MovePickCursor(0, step, 5)
	if cursor != 1 {
		t.Fatalf("move past end = %d, want 1", cursor)
	}
	answer, ok := PickAnswer(step, 99)
	if !ok {
		t.Fatal("pick answer unavailable")
	}
	if answer.Kind != wizard.KindPick || answer.ChoiceID != "anthropic" {
		t.Fatalf("answer = (%q, %q), want pick anthropic", answer.Kind, answer.ChoiceID)
	}

	_, ok = PickAnswer(wizard.Pick("empty", "Empty", nil), 0)
	if ok {
		t.Fatal("empty pick unexpectedly returned answer")
	}
}
