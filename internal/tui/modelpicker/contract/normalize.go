package contract

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/normalization"

// NormalizeProviderEntry trims a provider entry and fills an empty label from
// the provider ID. The bool is false when the provider has no stable ID.
func NormalizeProviderEntry(entry ProviderEntry) (ProviderEntry, bool) {
	return normalization.ProviderEntry(entry)
}

// NormalizeModelEntry trims a model entry and fills an empty label from the
// model ID. The bool is false when the model has no stable ID.
func NormalizeModelEntry(entry ModelEntry) (ModelEntry, bool) {
	return normalization.ModelEntry(entry)
}

// NormalizeModelEntries trims model entries, drops entries without stable IDs,
// and fills empty labels from model IDs.
func NormalizeModelEntries(entries []ModelEntry) []ModelEntry {
	return normalization.ModelEntries(entries)
}
