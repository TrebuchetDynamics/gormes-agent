package contract

import "strings"

// NormalizeProviderEntry trims a provider entry and fills an empty label from
// the provider ID. The bool is false when the provider has no stable ID.
func NormalizeProviderEntry(entry ProviderEntry) (ProviderEntry, bool) {
	id := strings.TrimSpace(entry.ID)
	if id == "" {
		return ProviderEntry{}, false
	}
	return ProviderEntry{ID: id, Label: firstNonEmptyString(strings.TrimSpace(entry.Label), id)}, true
}

// NormalizeModelEntry trims a model entry and fills an empty label from the
// model ID. The bool is false when the model has no stable ID.
func NormalizeModelEntry(entry ModelEntry) (ModelEntry, bool) {
	id := strings.TrimSpace(entry.ID)
	if id == "" {
		return ModelEntry{}, false
	}
	return ModelEntry{ID: id, Label: firstNonEmptyString(strings.TrimSpace(entry.Label), id)}, true
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
