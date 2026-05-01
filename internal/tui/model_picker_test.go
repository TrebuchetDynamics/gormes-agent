package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModelPicker_RenderProviderList proves the renderer shows all providers
// in a 2-column grid with the selected provider highlighted by "❯".
func TestModelPicker_RenderProviderList(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
			{ID: "openai", Label: "OpenAI"},
			{ID: "google", Label: "Google"},
			{ID: "ollama", Label: "Ollama"},
		},
		SelectedProviderIndex: 1, // OpenAI selected
		SelectedModelIndex:   -1,
	}

	got := RenderModelPicker(state)
	if got == "" {
		t.Fatal("RenderModelPicker returned empty string")
	}

	// All provider labels must appear
	for _, p := range state.Providers {
		if !strings.Contains(got, p.Label) {
			t.Fatalf("render missing provider label %q in %q", p.Label, got)
		}
	}

	// Selected provider (OpenAI) must be marked with "❯"
	lines := strings.Split(got, "\n")
	foundSelected := false
	for _, line := range lines {
		if strings.Contains(line, "❯") && strings.Contains(line, "OpenAI") {
			foundSelected = true
			break
		}
	}
	if !foundSelected {
		t.Fatalf("selected provider OpenAI not marked with ❯ in:\n%s", got)
	}

	// Verify that Anthropic and Google do NOT have ❯ directly before them.
	// In 2-column layout, a provider line contains both columns. We need to
	// check that the ❯ marker is NOT immediately adjacent to these labels.
	for _, line := range lines {
		// Check if Anthropic has ❯ prefix (should be "❯ Anthropic", not "  Anthropic" with ❯ elsewhere on line)
		// Since the line format is "  Anthropic      ❯ OpenAI", we check for "❯ " right before "Anthropic"
		if strings.Contains(line, "❯ Anthropic") || strings.Contains(line, "❯  Anthropic") {
			t.Fatalf("non-selected provider Anthropic should not have ❯ marker directly before it: %q", line)
		}
		if strings.Contains(line, "❯ Google") || strings.Contains(line, "❯  Google") {
			t.Fatalf("non-selected provider Google should not have ❯ marker directly before it: %q", line)
		}
	}
}

// TestModelPicker_RenderModelListWhenProviderSelected proves that when a
// provider is selected, the model list column appears with models for that
// provider and the current model is marked with "*".
func TestModelPicker_RenderModelListWhenProviderSelected(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   24,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
		Models: []ModelEntry{
			{ID: "claude-opus-4", Label: "Claude Opus 4"},
			{ID: "claude-sonnet-4", Label: "Claude Sonnet 4"},
			{ID: "claude-sonnet-3", Label: "Claude Sonnet 3"},
		},
		SelectedModelIndex: 1, // Sonnet 4 selected
		CurrentProvider:    "anthropic",
		CurrentModel:       "claude-sonnet-3", // Different from selected
	}

	got := RenderModelPicker(state)
	if got == "" {
		t.Fatal("RenderModelPicker returned empty string")
	}

	// All model labels must appear
	for _, m := range state.Models {
		if !strings.Contains(got, m.Label) {
			t.Fatalf("render missing model label %q in %q", m.Label, got)
		}
	}

	// Selected model (Sonnet 4) must be marked with "❯"
	lines := strings.Split(got, "\n")
	foundSelected := false
	for _, line := range lines {
		if strings.Contains(line, "❯") && strings.Contains(line, "Sonnet 4") {
			foundSelected = true
			break
		}
	}
	if !foundSelected {
		t.Fatalf("selected model Sonnet 4 not marked with ❯ in:\n%s", got)
	}

	// Current model (Sonnet 3) must be marked with "*"
	foundCurrent := false
	for _, line := range lines {
		if strings.Contains(line, "*") && strings.Contains(line, "Sonnet 3") {
			foundCurrent = true
			break
		}
	}
	if !foundCurrent {
		t.Fatalf("current model Sonnet 3 not marked with * in:\n%s", got)
	}
}

// TestModelPicker_RenderCurrentModelIndicator proves that when the selected
// model equals the current model, it shows "*❯" or "❯ *" combining both markers.
func TestModelPicker_RenderCurrentModelIndicator(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   24,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
		Models: []ModelEntry{
			{ID: "claude-opus-4", Label: "Claude Opus 4"},
			{ID: "claude-sonnet-4", Label: "Claude Sonnet 4"},
		},
		SelectedModelIndex: 1, // Sonnet 4 selected
		CurrentProvider:   "anthropic",
		CurrentModel:      "claude-sonnet-4", // Same as selected
	}

	got := RenderModelPicker(state)
	if got == "" {
		t.Fatal("RenderModelPicker returned empty string")
	}

	// The selected+current model (Sonnet 4) must show both "*" and "❯"
	lines := strings.Split(got, "\n")
	foundCombined := false
	for _, line := range lines {
		if strings.Contains(line, "Sonnet 4") {
			hasCurrent := strings.Contains(line, "*")
			hasSelected := strings.Contains(line, "❯")
			if hasCurrent && hasSelected {
				foundCombined = true
				break
			}
		}
	}
	if !foundCombined {
		t.Fatalf("current+selected model should show both * and ❯ markers in:\n%s", got)
	}
}

// TestModelPicker_RenderKeyboardHints proves the keyboard navigation hints
// appear at the bottom of the rendered picker.
func TestModelPicker_RenderKeyboardHints(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
	}

	got := RenderModelPicker(state)
	if got == "" {
		t.Fatal("RenderModelPicker returned empty string")
	}

	// Keyboard hints must appear
	hints := []string{"↑/↓", "navigate", "Enter", "confirm", "Esc", "cancel"}
	for _, hint := range hints {
		if !strings.Contains(got, hint) {
			t.Fatalf("render missing keyboard hint %q in:\n%s", hint, got)
		}
	}
}

// TestModelPicker_KeyboardNavUpDownProviderNavigation proves Up/Down navigate
// through the provider list when no provider is selected.
func TestModelPicker_KeyboardNavUpDownProviderNavigation(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
			{ID: "openai", Label: "OpenAI"},
			{ID: "google", Label: "Google"},
		},
		SelectedProviderIndex: 1, // OpenAI selected initially
		SelectedModelIndex:   -1,
	}

	// Press Up - should move to Anthropic
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyUp}, state)
	if state.SelectedProviderIndex != 0 {
		t.Fatalf("Up from index 1: got %d, want 0", state.SelectedProviderIndex)
	}

	// Press Down twice - should move to Google (index 2)
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyDown}, state)
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyDown}, state)
	if state.SelectedProviderIndex != 2 {
		t.Fatalf("Down x2 from index 0: got %d, want 2", state.SelectedProviderIndex)
	}

	// Press Down again - should stay at last item
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyDown}, state)
	if state.SelectedProviderIndex != 2 {
		t.Fatalf("Down at end: got %d, want 2 (stay at last)", state.SelectedProviderIndex)
	}
}

// TestModelPicker_KeyboardNavEnterConfirmsSelection proves Enter emits a
// modelPickerConfirmedMsg with the selected provider and model.
func TestModelPicker_KeyboardNavEnterConfirmsSelection(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
		Models: []ModelEntry{
			{ID: "claude-opus-4", Label: "Claude Opus 4"},
		},
		SelectedModelIndex: 0,
		CurrentProvider:   "other",
		CurrentModel:      "other-model",
	}

	var cmd tea.Cmd
	state, cmd = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyEnter}, state)

	if cmd == nil {
		t.Fatal("Enter should return a command")
	}

	msg := cmd()
	confirmed, ok := msg.(modelPickerConfirmedMsg)
	if !ok {
		t.Fatalf("expected modelPickerConfirmedMsg, got %T", msg)
	}

	if confirmed.Provider != "anthropic" {
		t.Fatalf("confirmed provider: got %q, want %q", confirmed.Provider, "anthropic")
	}
	if confirmed.Model != "claude-opus-4" {
		t.Fatalf("confirmed model: got %q, want %q", confirmed.Model, "claude-opus-4")
	}
}

// TestModelPicker_KeyboardNavEscapeCancels proves Escape emits an empty
// modelPickerConfirmedMsg to signal cancellation.
func TestModelPicker_KeyboardNavEscapeCancels(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
	}

	var cmd tea.Cmd
	state, cmd = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyEscape}, state)

	if cmd == nil {
		t.Fatal("Escape should return a command")
	}

	msg := cmd()
	confirmed, ok := msg.(modelPickerConfirmedMsg)
	if !ok {
		t.Fatalf("expected modelPickerConfirmedMsg, got %T", msg)
	}

	// Empty result signals cancel
	if confirmed.Provider != "" || confirmed.Model != "" {
		t.Fatalf("escape should produce empty result, got %+v", confirmed)
	}
}

// TestModelPicker_KeyboardNavRightEntersModelList proves that pressing Right
// when on a provider enters the model list and selects the first model.
func TestModelPicker_KeyboardNavRightEntersModelList(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
		Models: []ModelEntry{
			{ID: "claude-opus-4", Label: "Claude Opus 4"},
			{ID: "claude-sonnet-4", Label: "Claude Sonnet 4"},
		},
		SelectedModelIndex: -1, // Not in model list yet
	}

	// Press Right to enter model list
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyRight}, state)
	if state.SelectedModelIndex != 0 {
		t.Fatalf("Right should select first model: got %d, want 0", state.SelectedModelIndex)
	}

	// Press Left to go back to provider list
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyLeft}, state)
	if state.SelectedModelIndex != -1 {
		t.Fatalf("Left should exit model list: got %d, want -1", state.SelectedModelIndex)
	}
}

// TestModelPicker_KeyboardNavUpDownInModelList proves Up/Down navigate
// through the model list when a provider is selected and model list is active.
func TestModelPicker_KeyboardNavUpDownInModelList(t *testing.T) {
	state := ModelPickerState{
		Width:    60,
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
		},
		SelectedProviderIndex: 0,
		Models: []ModelEntry{
			{ID: "claude-opus-4", Label: "Claude Opus 4"},
			{ID: "claude-sonnet-4", Label: "Claude Sonnet 4"},
			{ID: "claude-sonnet-3", Label: "Claude Sonnet 3"},
		},
		SelectedModelIndex: 1, // Sonnet 4 selected
	}

	// Press Up - should move to Opus 4
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyUp}, state)
	if state.SelectedModelIndex != 0 {
		t.Fatalf("Up from index 1: got %d, want 0", state.SelectedModelIndex)
	}

	// Press Down - should move back to Sonnet 4
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyDown}, state)
	if state.SelectedModelIndex != 1 {
		t.Fatalf("Down from index 0: got %d, want 1", state.SelectedModelIndex)
	}

	// Press Down again - should move to Sonnet 3
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyDown}, state)
	if state.SelectedModelIndex != 2 {
		t.Fatalf("Down from index 1: got %d, want 2", state.SelectedModelIndex)
	}

	// Down at end - should stay at last
	state, _ = UpdateModelPicker(tea.KeyMsg{Type: tea.KeyDown}, state)
	if state.SelectedModelIndex != 2 {
		t.Fatalf("Down at end: got %d, want 2", state.SelectedModelIndex)
	}
}

// TestModelPicker_NarrowTerminalFallback proves the renderer handles narrow
// terminals (< 30 cols) gracefully with a single-column layout.
func TestModelPicker_NarrowTerminalFallback(t *testing.T) {
	state := ModelPickerState{
		Width:    25, // Narrow
		Height:   20,
		Providers: []ProviderEntry{
			{ID: "anthropic", Label: "Anthropic"},
			{ID: "openai", Label: "OpenAI"},
		},
		SelectedProviderIndex: 0,
	}

	got := RenderModelPicker(state)
	if got == "" {
		t.Fatal("RenderModelPicker returned empty string for narrow terminal")
	}

	// Should still show the providers
	if !strings.Contains(got, "Anthropic") || !strings.Contains(got, "OpenAI") {
		t.Fatalf("narrow render missing providers in:\n%s", got)
	}
}

// TestModelPicker_EmptyProvidersHandlesGracefully proves the renderer handles
// an empty provider list without panicking.
func TestModelPicker_EmptyProvidersHandlesGracefully(t *testing.T) {
	state := ModelPickerState{
		Width:               60,
		Height:              20,
		Providers:           []ProviderEntry{},
		SelectedProviderIndex: -1,
	}

	got := RenderModelPicker(state)
	if got == "" {
		t.Fatal("RenderModelPicker returned empty string for empty providers")
	}

	// Should still show title and hints
	if !strings.Contains(got, "Select Model") {
		t.Fatalf("missing title in empty providers render:\n%s", got)
	}
}
