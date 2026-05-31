package selection

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/identity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/schema"
)

// ModelListFocused reports whether keyboard navigation is currently inside
// the model list instead of the provider list.
func ModelListFocused(state schema.State) bool {
	return state.SelectedModelIndex >= 0
}

// MoveIndex moves a selected index within a list without wrapping.
func MoveIndex(index, delta, count int) int {
	if count <= 0 {
		return -1
	}
	if index < 0 {
		index = 0
	}
	index += delta
	if index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
}

// Provider returns the provider currently focused by the user.
func Provider(state schema.State) (schema.ProviderEntry, bool) {
	return selectedValue(state.Providers, state.SelectedProviderIndex)
}

// Model returns the model currently focused by the user.
func Model(state schema.State) (schema.ModelEntry, bool) {
	return selectedValue(state.Models, state.SelectedModelIndex)
}

// IsCurrentModel reports whether a provider/model row represents the current
// runtime selection shown by the picker.
func IsCurrentModel(state schema.State, providerID, modelID string) bool {
	return state.CurrentModel == modelID && identity.ProviderIDEqual(state.CurrentProvider, providerID)
}

// ConfirmedResult returns the provider/model pair emitted when the operator
// confirms the current focus. If no model row is focused, the active model is
// kept as the fallback for the selected provider.
func ConfirmedResult(state schema.State) (schema.Result, bool) {
	selectedProv, ok := Provider(state)
	if !ok {
		return schema.Result{}, false
	}
	selectedModel := state.CurrentModel
	if focusedModel, ok := Model(state); ok {
		selectedModel = focusedModel.ID
	}
	return schema.Result{Provider: selectedProv.ID, Model: selectedModel}, true
}

// NormalizeConfirmed returns a catalog-valid provider/model pair from a
// confirmed selection. Unknown models fall back to the first model for a known
// provider; unknown providers and explicitly empty models are preserved after
// trimming so callers can decide how to report or apply them.
func NormalizeConfirmed(catalog []schema.CatalogProvider, provider, model string) (string, string) {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	for _, entry := range catalog {
		if !identity.ProviderIDEqual(entry.Provider.ID, provider) {
			continue
		}
		if model != "" {
			for _, candidate := range entry.Models {
				if candidate.ID == model {
					return provider, model
				}
			}
		}
		if len(entry.Models) > 0 {
			return provider, entry.Models[0].ID
		}
	}
	return provider, model
}

func selectedValue[T any](values []T, index int) (T, bool) {
	if index < 0 || index >= len(values) {
		var zero T
		return zero, false
	}
	return values[index], true
}
