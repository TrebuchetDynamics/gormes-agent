package wizard

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestWizardChassis_TextStepCapturesInput(t *testing.T) {
	m := newModel([]Step{
		Text("name", "Agent name"),
	})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Type("operator")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if final.err != nil {
		t.Fatalf("wizard error = %v", final.err)
	}
	if got := final.result.String("name"); got != "operator" {
		t.Fatalf("result name = %q, want %q", got, "operator")
	}
}

func TestWizardChassis_PickerStepReturnsSelection(t *testing.T) {
	m := newModel([]Step{
		Pick("provider", "Provider", []Choice{
			{ID: "anthropic", Label: "Anthropic"},
			{ID: "openai", Label: "OpenAI"},
			{ID: "local", Label: "Local"},
		}),
	})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if final.err != nil {
		t.Fatalf("wizard error = %v", final.err)
	}
	if got := final.result.Choice("provider"); got != "openai" {
		t.Fatalf("provider choice = %q, want %q", got, "openai")
	}
}

func TestWizardChassis_PickerStepSupportsTmuxFriendlyKeys(t *testing.T) {
	m := newModel([]Step{
		Pick("mode", "Setup mode", []Choice{
			{ID: "quick", Label: "Quick setup"},
			{ID: "full", Label: "Full setup"},
			{ID: "exit", Label: "Exit"},
		}, WithDefaultChoice("full")),
	})

	view := m.View()
	for _, want := range []string{"1. Quick setup", "> 2. Full setup", "Up/Down", "j/k", "1-9 select", "Esc/q abort"} {
		if !strings.Contains(view, want) {
			t.Fatalf("picker view missing %q:\n%s", want, view)
		}
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if final.err != nil {
		t.Fatalf("wizard error = %v", final.err)
	}
	if got := final.result.Choice("mode"); got != "exit" {
		t.Fatalf("mode choice = %q, want direct numeric selection to choose exit", got)
	}
}

func TestWizardChassis_ConfirmStepHonorsKeybindings(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
		want bool
	}{
		{name: "yes", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, want: true},
		{name: "no", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel([]Step{
				Confirm("apply", "Apply changes?"),
			})
			tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

			tm.Send(tt.key)

			final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
			if final.err != nil {
				t.Fatalf("wizard error = %v", final.err)
			}
			if got := final.result.Bool("apply"); got != tt.want {
				t.Fatalf("apply confirm = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWizardChassis_NonInteractiveReturnsErrRequiresTTY(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()

	_, err = New(WithInput(devNull)).Run(context.Background(), Text("name", "Agent name"))
	if !errors.Is(err, ErrRequiresTTY) {
		t.Fatalf("Run error = %v, want ErrRequiresTTY", err)
	}
}

func TestWizardChassis_AbortReturnsErrAbort(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "ctrl-c", key: tea.KeyMsg{Type: tea.KeyCtrlC}},
		{name: "escape", key: tea.KeyMsg{Type: tea.KeyEscape}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel([]Step{
				Text("name", "Agent name"),
			})
			tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

			tm.Send(tt.key)

			final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
			if !errors.Is(final.err, ErrAbort) {
				t.Fatalf("wizard error = %v, want ErrAbort", final.err)
			}
		})
	}
}

func TestWizardChassis_PrefilledStepsBypassTTY(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()

	result, err := New(WithInput(devNull)).Run(context.Background(),
		Text("name", "Agent name", WithStringValue("prefilled")),
		Confirm("apply", "Apply changes?", WithBoolValue(true)),
	)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if got := result.String("name"); got != "prefilled" {
		t.Fatalf("name = %q, want prefilled", got)
	}
	if got := result.Bool("apply"); !got {
		t.Fatalf("apply = %v, want true", got)
	}
}
