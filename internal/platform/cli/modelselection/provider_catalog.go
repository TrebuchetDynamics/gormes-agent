package modelselection

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelcatalog"
)

const (
	ProviderCatalogAuxConfig      = modelcatalog.ProviderCatalogAuxConfig
	ProviderCatalogLeaveUnchanged = modelcatalog.ProviderCatalogLeaveUnchanged
)

type ProviderCatalogEntry = modelcatalog.ProviderEntry

func HermesProviderCatalog() []ProviderCatalogEntry {
	return modelcatalog.HermesProviderCatalog()
}

func HermesModelProviderCatalog() []ProviderCatalogEntry {
	return modelcatalog.HermesModelProviderCatalog()
}

func HermesModelProviderMenu() []ProviderMenuEntry {
	catalog := HermesModelProviderCatalog()
	entries := make([]ProviderMenuEntry, 0, len(catalog))
	for _, entry := range catalog {
		entries = append(entries, ProviderMenuEntry{ID: entry.ID, Label: entry.Label})
	}
	return entries
}

func HermesProviderCatalogMenu(activeProvider string) ([]ProviderMenuEntry, int) {
	catalog := HermesProviderCatalog()
	entries := make([]ProviderMenuEntry, 0, len(catalog))
	defaultIndex := len(catalog) - 1
	activeProvider = strings.TrimSpace(activeProvider)
	for i, entry := range catalog {
		label := entry.Label
		if activeProvider != "" && strings.TrimSpace(entry.ID) == activeProvider {
			label += "  ← currently active"
			defaultIndex = i
		}
		entries = append(entries, ProviderMenuEntry{ID: entry.ID, Label: label})
	}
	return entries, defaultIndex
}
