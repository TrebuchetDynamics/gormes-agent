package catalog

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type ProviderEntry = contract.ProviderEntry
type ModelEntry = contract.ModelEntry
type CatalogProvider = contract.CatalogProvider
type State = contract.State

func DefaultCatalog() ([]CatalogProvider, error) {
	providers := llm.ListPickerProviders()
	out := make([]CatalogProvider, 0, len(providers))
	for _, provider := range providers {
		providerEntry, ok := contract.NormalizeProviderEntry(ProviderEntry{ID: provider.Slug, Label: provider.Label})
		if !ok {
			continue
		}
		modelIDs := provider.Models
		if len(modelIDs) == 0 {
			modelIDs = llm.ProviderModelCatalogSuggestions(providerEntry.ID, nil)
		}
		models := contract.NormalizeModelEntries(modelEntriesFromIDs(modelIDs))
		if len(models) == 0 {
			continue
		}
		out = append(out, CatalogProvider{
			Provider: providerEntry,
			Models:   models,
		})
	}
	return out, nil
}

func modelEntriesFromIDs(modelIDs []string) []ModelEntry {
	entries := make([]ModelEntry, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		entries = append(entries, ModelEntry{ID: modelID, Label: modelID})
	}
	return entries
}

func SlashArgument(input string) string {
	return contract.SlashArgument(input)
}

func NormalizeCatalog(catalog []CatalogProvider) []CatalogProvider {
	out := make([]CatalogProvider, 0, len(catalog))
	for _, entry := range catalog {
		provider, ok := contract.NormalizeProviderEntry(entry.Provider)
		if !ok {
			continue
		}
		models := contract.NormalizeModelEntries(entry.Models)
		if len(models) == 0 {
			continue
		}
		out = append(out, CatalogProvider{
			Provider: provider,
			Models:   models,
		})
	}
	return out
}

func NewState(catalog []CatalogProvider, currentProvider, currentModel string, width, height int) State {
	providers := make([]ProviderEntry, 0, len(catalog))
	selectedProvider := 0
	for i, entry := range catalog {
		providers = append(providers, entry.Provider)
		if currentProvider != "" && contract.ProviderIDEqual(entry.Provider.ID, currentProvider) {
			selectedProvider = i
		}
	}
	models := ModelsForProviderIndex(catalog, selectedProvider)
	return State{
		Width:                 width,
		Height:                height,
		Providers:             providers,
		SelectedProviderIndex: selectedProvider,
		Models:                models,
		SelectedModelIndex:    -1,
		CurrentProvider:       currentProvider,
		CurrentModel:          currentModel,
	}
}

func ModelsForProviderIndex(catalog []CatalogProvider, idx int) []ModelEntry {
	if idx < 0 || idx >= len(catalog) {
		return nil
	}
	return append([]ModelEntry(nil), catalog[idx].Models...)
}

func NormalizeConfirmedSelection(catalog []CatalogProvider, provider, model string) (string, string) {
	return contract.NormalizeConfirmedSelection(catalog, provider, model)
}
