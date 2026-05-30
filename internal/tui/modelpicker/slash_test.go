package modelpicker

import "testing"

func TestSlashArgument(t *testing.T) {
	if got := SlashArgument("/model openai/gpt-4.1"); got != "openai/gpt-4.1" {
		t.Fatalf("SlashArgument = %q", got)
	}
	if got := SlashArgument("/model"); got != "" {
		t.Fatalf("SlashArgument no arg = %q", got)
	}
}

func TestNormalizeCatalogAndState(t *testing.T) {
	catalog := NormalizeCatalog([]CatalogProvider{
		{Provider: ProviderEntry{ID: " ", Label: "skip"}, Models: []ModelEntry{{ID: "x"}}},
		{Provider: ProviderEntry{ID: "openai"}, Models: []ModelEntry{{ID: " gpt-4.1 ", Label: " "}, {ID: ""}}},
		{Provider: ProviderEntry{ID: "empty"}},
	})
	if len(catalog) != 1 {
		t.Fatalf("NormalizeCatalog len = %d, want 1", len(catalog))
	}
	if catalog[0].Provider.Label != "openai" || catalog[0].Models[0].Label != "gpt-4.1" {
		t.Fatalf("NormalizeCatalog = %+v", catalog)
	}
	state := NewState(catalog, "OPENAI", "gpt-4.1", 80, 24)
	if state.SelectedProviderIndex != 0 || len(state.Models) != 1 || state.Width != 80 || state.Height != 24 {
		t.Fatalf("NewState = %+v", state)
	}
	models := ModelsForProviderIndex(catalog, 0)
	models[0].ID = "mutated"
	if catalog[0].Models[0].ID == "mutated" {
		t.Fatal("ModelsForProviderIndex returned catalog backing slice")
	}

	provider, model := NormalizeConfirmedSelection(catalog, "OPENAI", "missing")
	if provider != "OPENAI" || model != "gpt-4.1" {
		t.Fatalf("NormalizeConfirmedSelection fallback = (%q, %q), want OPENAI/gpt-4.1", provider, model)
	}
}
