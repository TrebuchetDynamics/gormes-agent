package contract

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/schema"

// ProviderEntry is one provider option in the picker catalog.
type ProviderEntry = schema.ProviderEntry

// ModelEntry is one model option shown after a provider is selected.
type ModelEntry = schema.ModelEntry

// CatalogProvider is the TUI-local provider/model catalog shape consumed by
// the /model overlay.
type CatalogProvider = schema.CatalogProvider

// State is the complete state for the model picker overlay. Width and Height
// carry the terminal dimensions for layout calculations.
type State = schema.State

// Result is returned when the user confirms a model selection. It carries the
// chosen provider and model IDs. An empty result signals cancellation.
type Result = schema.Result
