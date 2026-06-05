package contract

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/identity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/selection"
)

// ProviderIDEqual reports whether two provider IDs identify the same provider.
// Provider selection accepts case-insensitive IDs so slash-command input can
// preserve the operator's typed casing while still matching catalog entries.
func ProviderIDEqual(left, right string) bool {
	return identity.ProviderIDEqual(left, right)
}

// ModelListFocused reports whether keyboard navigation is currently inside
// the model list instead of the provider list.
func ModelListFocused(state State) bool {
	return selection.ModelListFocused(state)
}

// MoveSelection moves a selected index within a list without wrapping.
func MoveSelection(index, delta, count int) int {
	return selection.MoveIndex(index, delta, count)
}

// SelectedProvider returns the provider currently focused by the user.
func SelectedProvider(state State) (ProviderEntry, bool) {
	return selection.Provider(state)
}

// SelectedModel returns the model currently focused by the user.
func SelectedModel(state State) (ModelEntry, bool) {
	return selection.Model(state)
}

// IsCurrentModel reports whether a provider/model row represents the current
// runtime selection shown by the picker.
func IsCurrentModel(state State, providerID, modelID string) bool {
	return selection.IsCurrentModel(state, providerID, modelID)
}

// ConfirmedResult returns the provider/model pair emitted when the operator
// confirms the current focus. If no model row is focused, the active model is
// kept as the fallback for the selected provider.
func ConfirmedResult(state State) (Result, bool) {
	return selection.ConfirmedResult(state)
}

// NormalizeConfirmedSelection returns a catalog-valid provider/model pair from
// a confirmed selection. Unknown models fall back to the first model for a
// known provider; unknown providers and explicitly empty models are preserved
// after trimming so callers can decide how to report or apply them.
func NormalizeConfirmedSelection(catalog []CatalogProvider, provider, model string) (string, string) {
	return selection.NormalizeConfirmed(catalog, provider, model)
}
