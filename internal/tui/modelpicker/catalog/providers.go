package catalog

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelcatalog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract"
)

// HermesProviders adapts the shared Hermes-compatible provider catalog into
// the model picker row shape without exposing raw auth taxonomy labels.
func HermesProviders() []contract.ProviderEntry {
	catalog := modelcatalog.HermesModelProviderCatalog()
	entries := make([]contract.ProviderEntry, 0, len(catalog))
	for _, entry := range catalog {
		entries = append(entries, contract.ProviderEntry{ID: entry.ID, Label: entry.Label})
	}
	return entries
}
