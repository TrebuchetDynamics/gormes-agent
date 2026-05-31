package modelpicker

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/catalog"

// HermesProviders adapts the shared Hermes-compatible provider catalog into
// the model picker row shape without exposing raw auth taxonomy labels.
func HermesProviders() []ProviderEntry {
	return catalog.HermesProviders()
}
