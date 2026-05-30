package modelpicker

import tea "github.com/charmbracelet/bubbletea"

// ModelEntry is one model option shown after a provider is selected.
type ModelEntry struct {
	ID    string
	Label string
}

// State is the complete state for the model picker overlay. Width and Height
// carry the terminal dimensions for layout calculations.
type State struct {
	Width  int
	Height int

	// Providers is the list of available providers.
	Providers []ProviderEntry

	// SelectedProviderIndex is the 0-based index into Providers that is
	// currently focused by the user. -1 means no provider selected yet.
	SelectedProviderIndex int

	// Models is the list of models for the selected provider. Populated by the
	// caller after the provider is selected.
	Models []ModelEntry

	// SelectedModelIndex is the 0-based index into Models that is currently
	// focused. -1 means no model selected yet.
	SelectedModelIndex int

	// CurrentProvider is the provider ID of the currently active model. It is
	// used to mark the current model with "*" in the model list.
	CurrentProvider string

	// CurrentModel is the currently active model ID. It is marked with "*".
	CurrentModel string

	// Confirming is true when the user has pressed Enter on a model and the
	// picker should emit the confirmed selection.
	Confirming bool
}

// Result is returned when the user confirms a model selection. It carries the
// chosen provider and model IDs. An empty result signals cancellation.
type Result struct {
	Provider string
	Model    string
}

// Update handles keyboard events for the model picker state. The returned bool
// is true when result should be emitted to the caller.
func Update(msg tea.Msg, state State) (State, Result, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return updateKey(msg, state)
	}
	return state, Result{}, false
}

func updateKey(msg tea.KeyMsg, state State) (State, Result, bool) {
	switch msg.Type {
	case tea.KeyUp:
		if state.SelectedModelIndex >= 0 {
			if state.SelectedModelIndex > 0 {
				state.SelectedModelIndex--
			}
		} else if state.SelectedProviderIndex > 0 {
			state.SelectedProviderIndex--
		}
		return state, Result{}, false

	case tea.KeyDown:
		if state.SelectedModelIndex >= 0 {
			if state.SelectedModelIndex < len(state.Models)-1 {
				state.SelectedModelIndex++
			}
		} else if state.SelectedProviderIndex < len(state.Providers)-1 {
			state.SelectedProviderIndex++
		}
		return state, Result{}, false

	case tea.KeyLeft:
		if state.SelectedModelIndex >= 0 {
			state.SelectedModelIndex = -1
		}
		return state, Result{}, false

	case tea.KeyRight:
		if state.SelectedProviderIndex >= 0 && state.SelectedModelIndex < 0 && len(state.Models) > 0 {
			state.SelectedModelIndex = 0
		}
		return state, Result{}, false

	case tea.KeyEnter:
		if state.SelectedProviderIndex >= 0 {
			selectedProv := state.Providers[state.SelectedProviderIndex]
			selectedModel := state.CurrentModel
			if state.SelectedModelIndex >= 0 && state.SelectedModelIndex < len(state.Models) {
				selectedModel = state.Models[state.SelectedModelIndex].ID
			}
			return state, Result{Provider: selectedProv.ID, Model: selectedModel}, true
		}
		return state, Result{}, false

	case tea.KeyEscape:
		return state, Result{}, true
	}
	return state, Result{}, false
}
