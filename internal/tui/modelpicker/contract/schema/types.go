package schema

// ProviderEntry is one provider option in the picker catalog.
type ProviderEntry struct {
	ID    string
	Label string
}

// ModelEntry is one model option shown after a provider is selected.
type ModelEntry struct {
	ID    string
	Label string
}

// CatalogProvider is the TUI-local provider/model catalog shape consumed by
// the /model overlay.
type CatalogProvider struct {
	Provider ProviderEntry
	Models   []ModelEntry
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
