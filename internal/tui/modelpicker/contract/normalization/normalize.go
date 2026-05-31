package normalization

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/schema"
)

// ProviderEntry trims a provider entry and fills an empty label from the
// provider ID. The bool is false when the provider has no stable ID.
func ProviderEntry(entry schema.ProviderEntry) (schema.ProviderEntry, bool) {
	id, label, ok := normalizeIDLabel(entry.ID, entry.Label)
	if !ok {
		return schema.ProviderEntry{}, false
	}
	return schema.ProviderEntry{ID: id, Label: label}, true
}

// ModelEntry trims a model entry and fills an empty label from the model ID.
// The bool is false when the model has no stable ID.
func ModelEntry(entry schema.ModelEntry) (schema.ModelEntry, bool) {
	id, label, ok := normalizeIDLabel(entry.ID, entry.Label)
	if !ok {
		return schema.ModelEntry{}, false
	}
	return schema.ModelEntry{ID: id, Label: label}, true
}

// ModelEntries trims model entries, drops entries without stable IDs, and
// fills empty labels from model IDs.
func ModelEntries(entries []schema.ModelEntry) []schema.ModelEntry {
	out := make([]schema.ModelEntry, 0, len(entries))
	for _, entry := range entries {
		model, ok := ModelEntry(entry)
		if !ok {
			continue
		}
		out = append(out, model)
	}
	return out
}

func normalizeIDLabel(rawID, rawLabel string) (string, string, bool) {
	id := strings.TrimSpace(rawID)
	if id == "" {
		return "", "", false
	}
	return id, firstNonEmptyString(strings.TrimSpace(rawLabel), id), true
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
