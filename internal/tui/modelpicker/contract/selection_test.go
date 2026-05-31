package contract

import "testing"

func TestSelectedProvider(t *testing.T) {
	state := State{
		Providers:             []ProviderEntry{{ID: "anthropic", Label: "Anthropic"}},
		SelectedProviderIndex: 0,
	}
	provider, ok := SelectedProvider(state)
	if !ok || provider.ID != "anthropic" {
		t.Fatalf("SelectedProvider = (%#v, %v), want anthropic true", provider, ok)
	}

	state.SelectedProviderIndex = 1
	provider, ok = SelectedProvider(state)
	if ok || provider.ID != "" {
		t.Fatalf("SelectedProvider out of range = (%#v, %v), want zero false", provider, ok)
	}
}

func TestSelectedModel(t *testing.T) {
	state := State{
		Models:             []ModelEntry{{ID: "claude", Label: "Claude"}},
		SelectedModelIndex: 0,
	}
	model, ok := SelectedModel(state)
	if !ok || model.ID != "claude" {
		t.Fatalf("SelectedModel = (%#v, %v), want claude true", model, ok)
	}

	state.SelectedModelIndex = -1
	model, ok = SelectedModel(state)
	if ok || model.ID != "" {
		t.Fatalf("SelectedModel out of range = (%#v, %v), want zero false", model, ok)
	}
}
