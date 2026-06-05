package catalog

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelcatalog"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/menu"
)

const (
	ProviderCatalogAuxConfig      = modelcatalog.ProviderCatalogAuxConfig
	ProviderCatalogLeaveUnchanged = modelcatalog.ProviderCatalogLeaveUnchanged
)

type ProviderCatalogEntry = modelcatalog.ProviderEntry

type ProviderMenuEntry = menu.ProviderEntry

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
	for _, entry := range catalog {
		entries = append(entries, ProviderMenuEntry{ID: entry.ID, Label: entry.Label})
	}
	return menu.AnnotateCurrentProvider(entries, activeProvider, len(catalog)-1)
}
