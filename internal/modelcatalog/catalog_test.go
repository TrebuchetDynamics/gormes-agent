package modelcatalog

import "testing"

func TestHermesModelProviderCatalogExcludesSetupActions(t *testing.T) {
	entries := HermesModelProviderCatalog()
	if len(entries) != 37 {
		t.Fatalf("model provider entries = %d, want 37", len(entries))
	}
	for _, entry := range entries {
		switch entry.ID {
		case ProviderCatalogAuxConfig, ProviderCatalogLeaveUnchanged, "custom-endpoint":
			t.Fatalf("model provider catalog contains setup-only entry: %#v", entry)
		}
	}
}

func TestHermesProviderCatalogReturnsCopy(t *testing.T) {
	entries := HermesProviderCatalog()
	entries[0].ID = "mutated"

	if got := HermesProviderCatalog()[0].ID; got != "nous" {
		t.Fatalf("catalog first id = %q, want immutable copy", got)
	}
}
