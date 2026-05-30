package modelpicker

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelcatalog"

// ProviderEntry is one provider option in the picker catalog.
type ProviderEntry struct {
	ID    string
	Label string
}

// HermesProviders adapts the shared Hermes-compatible provider catalog into
// the model picker row shape without exposing raw auth taxonomy labels.
func HermesProviders() []ProviderEntry {
	catalog := modelcatalog.HermesModelProviderCatalog()
	entries := make([]ProviderEntry, 0, len(catalog))
	for _, entry := range catalog {
		entries = append(entries, ProviderEntry{ID: entry.ID, Label: entry.Label})
	}
	return entries
}
