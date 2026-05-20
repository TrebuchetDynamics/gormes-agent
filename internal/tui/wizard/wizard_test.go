package wizard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestWizardChassis_ViewHardeningBoundsSetupUX(t *testing.T) {
	long := strings.Repeat("x", 160)
	cases := []struct {
		name  string
		steps []Step
		want  []string
	}{
		{
			name:  "text input with long prompt placeholder and value",
			steps: []Step{Text("endpoint", "Gateway endpoint for a very long operator setup prompt "+long, WithPlaceholder("https://example.invalid/"+long), WithStringValue("https://127.0.0.1:8765/"+long))},
			want:  []string{"Gateway endpoint", "Enter submit"},
		},
		{
			name:  "password input masks long secret",
			steps: []Step{Password("api_key", "API key "+long, WithStringValue("sk-secret-"+long))},
			want:  []string{"API key", "Enter submit"},
		},
		{
			name:  "radio picker with long labels",
			steps: []Step{Pick("mode", "How should setup continue? "+long, []Choice{{ID: "quick", Label: "Quick setup recommended path " + long}, {ID: "full", Label: "Full setup advanced path " + long}}, WithRadioChoices())},
			want:  []string{"How should setup", "Quick setup", "Full setup"},
		},
		{
			name:  "numbered picker with long labels",
			steps: []Step{Pick("provider", "Provider selection "+long, []Choice{{ID: "anthropic", Label: "Anthropic Claude provider with detailed explanation " + long}, {ID: "openai", Label: "OpenAI provider with detailed explanation " + long}})},
			want:  []string{"Provider selection", "1. Anthropic", "2. OpenAI"},
		},
		{
			name:  "checklist with long labels",
			steps: []Step{Checklist("tools", "Tool setup "+long, []Choice{{ID: "browser", Label: "Browser automation toolset " + long}, {ID: "shell", Label: "Shell command toolset " + long}})},
			want:  []string{"Tool setup", "Browser automation", "Shell command"},
		},
		{
			name:  "confirm with long prompt",
			steps: []Step{Confirm("apply", "Apply these setup changes after reviewing this long warning "+long)},
			want:  []string{"Apply these", "No", "Yes"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, size := range []struct{ width, height int }{{20, 10}, {40, 12}, {80, 24}} {
				t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
					m := newModel(tc.steps)
					updated, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
					m = updated.(model)
					got := m.View()
					if strings.TrimSpace(got) == "" {
						t.Fatalf("wizard View returned blank output")
					}
					for _, line := range strings.Split(got, "\n") {
						if width := lipgloss.Width(line); width > size.width {
							t.Fatalf("wizard line width %d exceeds terminal width %d:\n%q\n\nfull output:\n%s", width, size.width, line, got)
						}
					}
					collapsed := strings.Join(strings.Fields(got), " ")
					for _, want := range tc.want {
						if !strings.Contains(collapsed, want) {
							t.Fatalf("wizard View missing %q:\n%s", want, got)
						}
					}
				})
			}
		})
	}
}

func TestWizardChassis_ViewHardeningBoundsShortSetupTerminal(t *testing.T) {
	long := strings.Repeat("x", 240)
	cases := []struct {
		name  string
		steps []Step
		want  []string
	}{
		{
			name:  "text input keeps prompt and help visible",
			steps: []Step{Text("endpoint", "Gateway endpoint "+long, WithStringValue("https://127.0.0.1:8765/"+long))},
			want:  []string{"Gateway endpoint", "omitted", "resize", "Enter submit"},
		},
		{
			name:  "picker keeps selected option and help visible",
			steps: []Step{Pick("provider", "Provider selection "+long, []Choice{{ID: "anthropic", Label: "Anthropic Claude provider " + long}, {ID: "openai", Label: "OpenAI provider " + long}})},
			want:  []string{"Provider selection", "omitted", "resize", "1. Anthropic", "Enter submit"},
		},
		{
			name:  "checklist keeps selected option and help visible",
			steps: []Step{Checklist("tools", "Tool setup "+long, []Choice{{ID: "browser", Label: "Browser automation toolset " + long}, {ID: "shell", Label: "Shell command toolset " + long}})},
			want:  []string{"Tool setup", "omitted", "resize", "Browser", "ENTER confirm"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel(tc.steps)
			updated, _ := m.Update(tea.WindowSizeMsg{Width: 24, Height: 8})
			m = updated.(model)
			got := m.View()
			lines := strings.Split(got, "\n")
			if len(lines) > 8 {
				t.Fatalf("wizard View height = %d, want <= 8:\n%s", len(lines), got)
			}
			for _, line := range lines {
				if width := lipgloss.Width(line); width > 24 {
					t.Fatalf("wizard line width %d exceeds terminal width 24:\n%q\n\nfull output:\n%s", width, line, got)
				}
			}
			collapsed := strings.Join(strings.Fields(got), " ")
			for _, want := range tc.want {
				if !strings.Contains(collapsed, want) {
					t.Fatalf("wizard short View missing %q:\n%s", want, got)
				}
			}
		})
	}
}

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

func TestWizardChassis_RadioPickerMatchesHermesSetupShape(t *testing.T) {
	m := newModel([]Step{
		Pick("mode", "How would you like to set up Gormes?", []Choice{
			{ID: "quick", Label: "Quick setup — provider, model & messaging (recommended)"},
			{ID: "full", Label: "Full setup — configure everything"},
		}, WithDefaultChoice("quick"), WithRadioChoices()),
	})

	view := m.View()
	for _, want := range []string{
		"How would you like to set up Gormes?",
		"→ (●) Quick setup — provider, model & messaging (recommended)",
		"(○) Full setup — configure everything",
		"↑↓ navigate  ENTER/SPACE select  ESC cancel",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("radio picker view missing %q:\n%s", want, view)
		}
	}
	for _, forbidden := range []string{"Gormes setup 1/1", "1. Quick setup", "1-9 select"} {
		if strings.Contains(view, forbidden) {
			t.Fatalf("radio picker should match Hermes-style first-run prompt and omit %q:\n%s", forbidden, view)
		}
	}
}

func TestWizardChassis_SingleStepPickerOmitsProgressLine(t *testing.T) {
	m := newModel([]Step{
		Pick("provider", "Provider", []Choice{{ID: "openai", Label: "OpenAI"}}),
	})
	if view := m.View(); strings.Contains(view, "Gormes setup 1/1") {
		t.Fatalf("single-step picker should omit progress line:\n%s", view)
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
