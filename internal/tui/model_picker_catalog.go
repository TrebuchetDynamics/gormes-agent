package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelcatalog"

// HermesModelPickerProviders adapts the shared Hermes-compatible provider
// catalog into the TUI picker row shape.
func HermesModelPickerProviders() []ProviderEntry {
	catalog := modelcatalog.HermesModelProviderCatalog()
	entries := make([]ProviderEntry, 0, len(catalog))
	for _, entry := range catalog {
		entries = append(entries, ProviderEntry{ID: entry.ID, Label: entry.Label})
	}
	return entries
}
