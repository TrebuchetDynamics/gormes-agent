package tuiapp

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOfflineTUIWiresLocalSmokeSubmit(t *testing.T) {
	setupNativeTUITestEnv(t)

	var rendered string
	cmd := newRootCommandWithRuntime(Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			return scriptedTUIProgram{run: func() {
				current := model
				current, _ = current.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
				for _, r := range "hello offline" {
					current, _ = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
				}
				current, updateCmd := current.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if updateCmd != nil {
					if msg := updateCmd(); msg != nil {
						current, _ = current.Update(msg)
					}
				}
				rendered = current.View()
			}}
		},
	})

	stdout, stderr, err := executeNativeTUICommand(cmd, "--offline")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"hello offline",
		"No provider call",
		"offline",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("offline TUI render missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "connect: connection refused") || strings.Contains(stderr, "api_server") {
		t.Fatalf("offline TUI surfaced provider/api_server failure\nrendered=%s\nstderr=%s", rendered, stderr)
	}
}

type scriptedTUIProgram struct {
	run func()
}

func (p scriptedTUIProgram) Run() (tea.Model, error) {
	if p.run != nil {
		p.run()
	}
	return nil, nil
}

func (p scriptedTUIProgram) Quit() {}
