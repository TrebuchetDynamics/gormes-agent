package modelpicker

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/navigation"
)

// Update handles keyboard events for the model picker state. The returned bool
// is true when result should be emitted to the caller.
func Update(msg tea.Msg, state State) (State, Result, bool) {
	return navigation.Update(msg, state)
}
