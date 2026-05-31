package wizardflow

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"

// Flow tracks shared wizard step progression for embedded admin screens.
// Rendering and per-step input widgets remain owned by the concrete screen.
type Flow struct {
	steps   []wizard.Step
	answers map[string]wizard.Answer
	index   int
}

// New returns a Flow over a defensive copy of steps.
func New(steps []wizard.Step) *Flow {
	return &Flow{
		steps:   append([]wizard.Step(nil), steps...),
		answers: map[string]wizard.Answer{},
	}
}

// ActiveStep returns the currently focused wizard step.
func (f *Flow) ActiveStep() (wizard.Step, bool) {
	if f == nil || f.index < 0 || f.index >= len(f.steps) {
		return wizard.Step{}, false
	}
	return f.steps[f.index], true
}

// Finish records an answer for the active step and advances to the next step.
// The returned bool is true when the flow has advanced past the final step.
func (f *Flow) Finish(answer wizard.Answer) bool {
	step, ok := f.ActiveStep()
	if !ok {
		return true
	}
	if answer.Kind == "" {
		answer.Kind = step.Kind
	}
	f.answers[step.ID] = answer
	f.index++
	return f.index >= len(f.steps)
}

// Index returns the zero-based active step index.
func (f *Flow) Index() int {
	if f == nil {
		return 0
	}
	return f.index
}

// Len returns the number of steps in the flow.
func (f *Flow) Len() int {
	if f == nil {
		return 0
	}
	return len(f.steps)
}

// Answer returns the recorded answer for id.
func (f *Flow) Answer(id string) wizard.Answer {
	if f == nil || f.answers == nil {
		return wizard.Answer{}
	}
	return f.answers[id]
}

// Text returns the recorded text answer for id.
func (f *Flow) Text(id string) string { return f.Answer(id).Text }

// Choice returns the recorded choice ID for id.
func (f *Flow) Choice(id string) string { return f.Answer(id).ChoiceID }

// Bool returns the recorded confirmation answer for id.
func (f *Flow) Bool(id string) bool { return f.Answer(id).Confirmed }
