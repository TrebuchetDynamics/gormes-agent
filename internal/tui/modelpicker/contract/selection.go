package contract

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
