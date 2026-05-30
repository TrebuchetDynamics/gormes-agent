package modelpicker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateNavigatesProvidersAndModels(t *testing.T) {
	state := State{
		Providers:             []ProviderEntry{{ID: "anthropic", Label: "Anthropic"}, {ID: "openai", Label: "OpenAI"}},
		Models:                []ModelEntry{{ID: "claude", Label: "Claude"}, {ID: "gpt", Label: "GPT"}},
		SelectedProviderIndex: 1,
		SelectedModelIndex:    -1,
	}
	state, _, emit := Update(tea.KeyMsg{Type: tea.KeyUp}, state)
	if emit || state.SelectedProviderIndex != 0 {
		t.Fatalf("up provider = index %d emit %v, want index 0 emit false", state.SelectedProviderIndex, emit)
	}
	state, _, emit = Update(tea.KeyMsg{Type: tea.KeyRight}, state)
	if emit || state.SelectedModelIndex != 0 {
		t.Fatalf("right = model index %d emit %v, want index 0 emit false", state.SelectedModelIndex, emit)
	}
	state, _, emit = Update(tea.KeyMsg{Type: tea.KeyDown}, state)
	if emit || state.SelectedModelIndex != 1 {
		t.Fatalf("down model = index %d emit %v, want index 1 emit false", state.SelectedModelIndex, emit)
	}
	state, _, emit = Update(tea.KeyMsg{Type: tea.KeyLeft}, state)
	if emit || state.SelectedModelIndex != -1 {
		t.Fatalf("left = model index %d emit %v, want -1 emit false", state.SelectedModelIndex, emit)
	}
}

func TestUpdateEmitsConfirmAndCancelResults(t *testing.T) {
	state := State{
		Providers:             []ProviderEntry{{ID: "anthropic", Label: "Anthropic"}},
		Models:                []ModelEntry{{ID: "claude", Label: "Claude"}},
		SelectedProviderIndex: 0,
		SelectedModelIndex:    0,
		CurrentModel:          "fallback",
	}
	_, result, emit := Update(tea.KeyMsg{Type: tea.KeyEnter}, state)
	if !emit || result.Provider != "anthropic" || result.Model != "claude" {
		t.Fatalf("enter result = %#v emit %v, want anthropic/claude emit true", result, emit)
	}
	_, result, emit = Update(tea.KeyMsg{Type: tea.KeyEscape}, state)
	if !emit || result.Provider != "" || result.Model != "" {
		t.Fatalf("escape result = %#v emit %v, want empty emit true", result, emit)
	}
}
