package contract

import "testing"

func TestNormalizeProviderEntry(t *testing.T) {
	provider, ok := NormalizeProviderEntry(ProviderEntry{ID: " openai ", Label: " "})
	if !ok || provider.ID != "openai" || provider.Label != "openai" {
		t.Fatalf("NormalizeProviderEntry = (%#v, %v), want openai/openai true", provider, ok)
	}
	provider, ok = NormalizeProviderEntry(ProviderEntry{ID: " ", Label: "skip"})
	if ok || provider.ID != "" || provider.Label != "" {
		t.Fatalf("NormalizeProviderEntry blank = (%#v, %v), want zero false", provider, ok)
	}
}

func TestNormalizeModelEntry(t *testing.T) {
	model, ok := NormalizeModelEntry(ModelEntry{ID: " gpt-4.1 ", Label: " GPT 4.1 "})
	if !ok || model.ID != "gpt-4.1" || model.Label != "GPT 4.1" {
		t.Fatalf("NormalizeModelEntry = (%#v, %v), want gpt-4.1/GPT 4.1 true", model, ok)
	}
	model, ok = NormalizeModelEntry(ModelEntry{ID: "", Label: "skip"})
	if ok || model.ID != "" || model.Label != "" {
		t.Fatalf("NormalizeModelEntry blank = (%#v, %v), want zero false", model, ok)
	}
}
