package contract

// SelectedModel returns the model currently focused by the user.
func SelectedModel(state State) (ModelEntry, bool) {
	if state.SelectedModelIndex < 0 || state.SelectedModelIndex >= len(state.Models) {
		return ModelEntry{}, false
	}
	return state.Models[state.SelectedModelIndex], true
}
