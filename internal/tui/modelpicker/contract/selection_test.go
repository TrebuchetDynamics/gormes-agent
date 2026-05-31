package contract

import "testing"

func TestModelListFocused(t *testing.T) {
	if ModelListFocused(State{SelectedModelIndex: -1}) {
		t.Fatal("ModelListFocused = true for provider focus")
	}
	if !ModelListFocused(State{SelectedModelIndex: 0}) {
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
			if got := MoveSelection(tt.index, tt.delta, tt.count); got != tt.want {
				t.Fatalf("MoveSelection(%d, %d, %d) = %d, want %d", tt.index, tt.delta, tt.count, got, tt.want)
			}
		})
	}
}

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
