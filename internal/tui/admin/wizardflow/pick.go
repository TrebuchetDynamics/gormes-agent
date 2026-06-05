package wizardflow

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/navigation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

// MovePickCursor applies vertical movement for a wizard pick step and clamps
// the cursor to available choices.
func MovePickCursor(cursor int, step wizard.Step, delta int) int {
	return navigation.MoveIndex(cursor, len(step.Choices), delta)
}

// PickAnswer returns the typed answer for the current cursor in a pick step.
func PickAnswer(step wizard.Step, cursor int) (wizard.Answer, bool) {
	if len(step.Choices) == 0 {
		return wizard.Answer{}, false
	}
	cursor = navigation.ClampIndex(cursor, len(step.Choices))
	return wizard.Answer{Kind: step.Kind, ChoiceID: step.Choices[cursor].ID}, true
}
