package contract

// ModelListFocused reports whether keyboard navigation is currently inside
// the model list instead of the provider list.
func ModelListFocused(state State) bool {
	return state.SelectedModelIndex >= 0
}

// MoveSelection moves a selected index within a list without wrapping.
func MoveSelection(index, delta, count int) int {
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

// SelectedProvider returns the provider currently focused by the user.
func SelectedProvider(state State) (ProviderEntry, bool) {
	if state.SelectedProviderIndex < 0 || state.SelectedProviderIndex >= len(state.Providers) {
		return ProviderEntry{}, false
	}
	return state.Providers[state.SelectedProviderIndex], true
}

// SelectedModel returns the model currently focused by the user.
func SelectedModel(state State) (ModelEntry, bool) {
	if state.SelectedModelIndex < 0 || state.SelectedModelIndex >= len(state.Models) {
		return ModelEntry{}, false
	}
	return state.Models[state.SelectedModelIndex], true
}
