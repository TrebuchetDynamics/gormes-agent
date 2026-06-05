package modelpicker

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/catalog"

func DefaultCatalog() ([]CatalogProvider, error) {
	return catalog.DefaultCatalog()
}

func SlashArgument(input string) string {
	return catalog.SlashArgument(input)
}

func NormalizeCatalog(entries []CatalogProvider) []CatalogProvider {
	return catalog.NormalizeCatalog(entries)
}

func NewState(entries []CatalogProvider, currentProvider, currentModel string, width, height int) State {
	return catalog.NewState(entries, currentProvider, currentModel, width, height)
}

func ModelsForProviderIndex(entries []CatalogProvider, idx int) []ModelEntry {
	return catalog.ModelsForProviderIndex(entries, idx)
}

func NormalizeConfirmedSelection(entries []CatalogProvider, provider, model string) (string, string) {
	return catalog.NormalizeConfirmedSelection(entries, provider, model)
}
