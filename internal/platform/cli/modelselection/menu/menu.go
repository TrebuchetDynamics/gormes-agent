package menu

import "strings"

// ProviderEntry is one provider option rendered by model-selection menus.
// Label is user-visible; ID is the canonical provider identifier persisted on select.
type ProviderEntry struct {
	ID    string
	Label string
}

// AnnotateCurrentProvider returns a copy of providers with the active provider
// label annotated and the matching default index. If no active provider is
// present, defaultIndex is fallbackDefault.
func AnnotateCurrentProvider(providers []ProviderEntry, activeProvider string, fallbackDefault int) ([]ProviderEntry, int) {
	entries := append([]ProviderEntry(nil), providers...)
	defaultIndex := fallbackDefault
	activeProvider = strings.TrimSpace(activeProvider)
	for i := range entries {
		if activeProvider != "" && strings.TrimSpace(entries[i].ID) == activeProvider {
			entries[i].Label = strings.TrimSpace(entries[i].Label) + "  ← currently active"
			defaultIndex = i
		}
	}
	return entries, defaultIndex
}
