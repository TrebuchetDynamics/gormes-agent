package navigation

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract"
)

// Update handles keyboard events for the model picker state. The returned bool
// is true when result should be emitted to the caller.
func Update(msg tea.Msg, state contract.State) (contract.State, contract.Result, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return updateKey(msg, state)
	}
	return state, contract.Result{}, false
}

func updateKey(msg tea.KeyMsg, state contract.State) (contract.State, contract.Result, bool) {
	switch msg.Type {
	case tea.KeyUp:
		if contract.ModelListFocused(state) {
			state.SelectedModelIndex = contract.MoveSelection(state.SelectedModelIndex, -1, len(state.Models))
		} else {
			state.SelectedProviderIndex = contract.MoveSelection(state.SelectedProviderIndex, -1, len(state.Providers))
		}
		return state, contract.Result{}, false

	case tea.KeyDown:
		if contract.ModelListFocused(state) {
			state.SelectedModelIndex = contract.MoveSelection(state.SelectedModelIndex, 1, len(state.Models))
		} else {
			state.SelectedProviderIndex = contract.MoveSelection(state.SelectedProviderIndex, 1, len(state.Providers))
		}
		return state, contract.Result{}, false

	case tea.KeyLeft:
		if contract.ModelListFocused(state) {
			state.SelectedModelIndex = -1
		}
		return state, contract.Result{}, false

	case tea.KeyRight:
		if state.SelectedProviderIndex >= 0 && !contract.ModelListFocused(state) && len(state.Models) > 0 {
			state.SelectedModelIndex = 0
		}
		return state, contract.Result{}, false

	case tea.KeyEnter:
		if selectedProv, ok := contract.SelectedProvider(state); ok {
			selectedModel := state.CurrentModel
			if focusedModel, ok := contract.SelectedModel(state); ok {
				selectedModel = focusedModel.ID
			}
			return state, contract.Result{Provider: selectedProv.ID, Model: selectedModel}, true
		}
		return state, contract.Result{}, false

	case tea.KeyEscape:
		return state, contract.Result{}, true
	}
	return state, contract.Result{}, false
}
