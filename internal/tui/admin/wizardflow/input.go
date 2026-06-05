package wizardflow

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputOptions describes the reusable text-entry contract for admin wizard
// steps. Concrete screens still own prompts, defaults, sizing, and validation.
type InputOptions struct {
	Prompt   string
	Width    int
	Value    string
	Password bool
}

// NewInput returns a focused Bubble Tea text input configured for an admin
// wizard step.
func NewInput(opts InputOptions) textinput.Model {
	input := textinput.New()
	input.Focus()
	input.Prompt = opts.Prompt
	if input.Prompt == "" {
		input.Prompt = "> "
	}
	if opts.Width > 0 {
		input.Width = opts.Width
	}
	if opts.Password {
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '*'
	}
	if opts.Value != "" {
		input.SetValue(opts.Value)
		input.CursorEnd()
	}
	return input
}

// UpdateInput applies one key message to an embedded wizard input. Existing
// admin wizard states synchronously drain textinput commands because they do
// not return those commands to the shell while focused.
func UpdateInput(input textinput.Model, msg tea.KeyMsg) textinput.Model {
	updated, cmd := input.Update(msg)
	if cmd != nil {
		_ = cmd()
	}
	return updated
}
