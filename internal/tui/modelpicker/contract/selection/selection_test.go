package selection

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/schema"
)

func TestModelListFocused(t *testing.T) {
	if ModelListFocused(schema.State{SelectedModelIndex: -1}) {
		t.Fatal("ModelListFocused = true for provider focus")
	}
	if !ModelListFocused(schema.State{SelectedModelIndex: 0}) {
		t.Fatal("ModelListFocused = false for model focus")
	}
}

func TestMoveSelectionClampsWithoutWrapping(t *testing.T) {
	tests := []struct {
		name  string
		index int
		delta int
		count int
		want  int
	}{
		{name: "up clamps at first", index: 0, delta: -1, count: 3, want: 0},
		{name: "down clamps at last", index: 2, delta: 1, count: 3, want: 2},
		{name: "moves inside range", index: 1, delta: -1, count: 3, want: 0},
		{name: "negative starts at first before moving", index: -1, delta: 1, count: 3, want: 1},
		{name: "empty list stays unfocused", index: 0, delta: 1, count: 0, want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MoveIndex(tt.index, tt.delta, tt.count); got != tt.want {
				t.Fatalf("MoveSelection(%d, %d, %d) = %d, want %d", tt.index, tt.delta, tt.count, got, tt.want)
			}
		})
	}
}

func TestSelectedProvider(t *testing.T) {
	state := schema.State{
		Providers:             []schema.ProviderEntry{{ID: "anthropic", Label: "Anthropic"}},
		SelectedProviderIndex: 0,
	}
	provider, ok := Provider(state)
	if !ok || provider.ID != "anthropic" {
		t.Fatalf("SelectedProvider = (%#v, %v), want anthropic true", provider, ok)
	}

	state.SelectedProviderIndex = 1
	provider, ok = Provider(state)
	if ok || provider.ID != "" {
		t.Fatalf("SelectedProvider out of range = (%#v, %v), want zero false", provider, ok)
	}
}

func TestSelectedModel(t *testing.T) {
	state := schema.State{
		Models:             []schema.ModelEntry{{ID: "claude", Label: "Claude"}},
		SelectedModelIndex: 0,
	}
	model, ok := Model(state)
	if !ok || model.ID != "claude" {
		t.Fatalf("SelectedModel = (%#v, %v), want claude true", model, ok)
	}

	state.SelectedModelIndex = -1
	model, ok = Model(state)
	if ok || model.ID != "" {
		t.Fatalf("SelectedModel out of range = (%#v, %v), want zero false", model, ok)
	}
}

func TestIsCurrentModelUsesProviderIdentityPolicy(t *testing.T) {
	state := schema.State{CurrentProvider: "OPENAI", CurrentModel: "gpt-4.1"}
	if !IsCurrentModel(state, "openai", "gpt-4.1") {
		t.Fatal("IsCurrentModel should use provider ID identity policy")
	}
	if IsCurrentModel(state, "openai", "gpt-4o") {
		t.Fatal("IsCurrentModel matched a different model")
	}
	if IsCurrentModel(state, "anthropic", "gpt-4.1") {
		t.Fatal("IsCurrentModel matched a different provider")
	}
}

func TestConfirmedResultUsesFocusedModelOrCurrentFallback(t *testing.T) {
	state := schema.State{
		Providers:             []schema.ProviderEntry{{ID: "anthropic", Label: "Anthropic"}},
		Models:                []schema.ModelEntry{{ID: "claude", Label: "Claude"}},
		SelectedProviderIndex: 0,
		SelectedModelIndex:    0,
		CurrentModel:          "fallback",
	}
	result, ok := ConfirmedResult(state)
	if !ok || result.Provider != "anthropic" || result.Model != "claude" {
		t.Fatalf("ConfirmedResult focused = (%#v, %v), want anthropic/claude true", result, ok)
	}

	state.SelectedModelIndex = -1
	result, ok = ConfirmedResult(state)
	if !ok || result.Provider != "anthropic" || result.Model != "fallback" {
		t.Fatalf("ConfirmedResult fallback = (%#v, %v), want anthropic/fallback true", result, ok)
	}

	state.SelectedProviderIndex = -1
	result, ok = ConfirmedResult(state)
	if ok || result.Provider != "" || result.Model != "" {
		t.Fatalf("ConfirmedResult no provider = (%#v, %v), want zero false", result, ok)
	}
}

func TestNormalizeConfirmedUsesCatalogPolicy(t *testing.T) {
	catalog := []schema.CatalogProvider{
		{
			Provider: schema.ProviderEntry{ID: "openai"},
			Models:   []schema.ModelEntry{{ID: "gpt-4.1"}, {ID: "o3"}},
		},
	}

	provider, model := NormalizeConfirmed(catalog, " OPENAI ", " o3 ")
	if provider != "OPENAI" || model != "o3" {
		t.Fatalf("NormalizeConfirmed known model = (%q, %q), want OPENAI/o3", provider, model)
	}

	provider, model = NormalizeConfirmed(catalog, " OPENAI ", " missing ")
	if provider != "OPENAI" || model != "gpt-4.1" {
		t.Fatalf("NormalizeConfirmed fallback = (%q, %q), want OPENAI/gpt-4.1", provider, model)
	}

	provider, model = NormalizeConfirmed(catalog, " unknown ", " custom ")
	if provider != "unknown" || model != "custom" {
		t.Fatalf("NormalizeConfirmed unknown provider = (%q, %q), want unknown/custom", provider, model)
	}
}
