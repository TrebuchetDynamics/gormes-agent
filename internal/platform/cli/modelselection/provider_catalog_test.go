package modelselection

import "testing"

func TestHermesProviderCatalogMenuMatchesSetupPickerContract(t *testing.T) {
	entries, defaultIndex := HermesProviderCatalogMenu("openai-codex")
	if len(entries) != 40 {
		t.Fatalf("provider entries = %d, want 40", len(entries))
	}
	want := []struct {
		index int
		id    string
		label string
	}{
		{0, "nous", "Nous Portal (Nous Research subscription)"},
		{5, "openai-codex", "OpenAI Codex  ← currently active"},
		{38, ProviderCatalogAuxConfig, "Configure auxiliary models..."},
		{39, ProviderCatalogLeaveUnchanged, "Leave unchanged"},
	}
	for _, item := range want {
		if entries[item.index].ID != item.id || entries[item.index].Label != item.label {
			t.Fatalf("entry[%d] = %#v, want id=%q label=%q", item.index, entries[item.index], item.id, item.label)
		}
	}
	if defaultIndex != 5 {
		t.Fatalf("default index = %d, want active provider index 5", defaultIndex)
	}
}

func TestHermesProviderCatalogMenuDefaultsToLeaveUnchangedWithoutActiveProvider(t *testing.T) {
	entries, defaultIndex := HermesProviderCatalogMenu("")
	if defaultIndex != len(entries)-1 {
		t.Fatalf("default index = %d, want Leave unchanged index %d", defaultIndex, len(entries)-1)
	}
	if got := entries[defaultIndex].ID; got != ProviderCatalogLeaveUnchanged {
		t.Fatalf("default entry id = %q, want %q", got, ProviderCatalogLeaveUnchanged)
	}
}

func TestHermesModelProviderCatalogExcludesSetupActions(t *testing.T) {
	entries := HermesModelProviderMenu()
	if len(entries) != 37 {
		t.Fatalf("model provider entries = %d, want 37", len(entries))
	}
	if got := entries[0]; got.ID != "nous" || got.Label != "Nous Portal (Nous Research subscription)" {
		t.Fatalf("first model provider = %#v", got)
	}
	if got := entries[5]; got.ID != "openai-codex" || got.Label != "OpenAI Codex" {
		t.Fatalf("sixth model provider = %#v", got)
	}
	for _, entry := range entries {
		switch entry.ID {
		case ProviderCatalogAuxConfig, ProviderCatalogLeaveUnchanged, "custom-endpoint":
			t.Fatalf("model provider catalog contains setup-only entry: %#v", entry)
		}
	}
}
