package providerregistry

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/modelcatalog"
)

// PickerProvider is one curated provider with its models array for the shared
// gateway, setup, CLI, and TUI model pickers.
type PickerProvider struct {
	// Slug is the canonical provider ID used in callback data (e.g. "openrouter").
	Slug string
	// Label is the display name in picker buttons.
	Label string
	// Index maps model indices to model IDs for callback data.
	Models []string
}

// ListPickerProviders returns the curated provider list for the shared
// model pickers. It filters the provider manifest to only
// implemented and owned providers, skipping row_backed and excluded
// entries. Aggregator providers (openrouter, etc.) are included so users
// can switch to an aggregator directly.
//
// This mirrors Hermes' list_picker_providers() which filters the
// upstream provider register list_picker_providers function.
func ListPickerProviders() []PickerProvider {
	manifest := HermesProviderRegistryManifest()
	out := make([]PickerProvider, 0, len(manifest))
	for _, entry := range manifest {
		switch entry.ImplementationStatus {
		case ProviderImplemented, ProviderOwned:
			// included
		default:
			continue
		}
		slug := strings.ToLower(strings.TrimSpace(entry.ID))
		if slug == "" {
			continue
		}
		label := pickerProviderDisplayLabel(entry)
		out = append(out, PickerProvider{
			Slug:   slug,
			Label:  label,
			Models: pickerProviderModelIDs(slug),
		})
	}
	return out
}

func pickerProviderModelIDs(provider string) []string {
	models := modelcatalog.ProviderModelCatalogSuggestions(provider, nil)
	if len(models) == 0 {
		return nil
	}
	return append([]string(nil), models...)
}

// pickerProviderDisplayLabel returns the human-readable label for a
// provider entry shown in the picker keyboard.
func pickerProviderDisplayLabel(entry ProviderManifestEntry) string {
	label := strings.Title(strings.ReplaceAll(
		strings.ReplaceAll(entry.ID, "-", " "),
		"_", " ",
	))
	label = strings.TrimSpace(label)
	if label == "" {
		label = entry.ID
	}
	return label
}
