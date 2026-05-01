// Package tui — Hermes-compatible ModelPicker overlay renderer and updater.
//
// ModelPickerState + RenderModelPicker implement the 2-step provider→model
// selection overlay that upstream Hermes exposes via modelPicker.tsx. The Go
// port is a pure renderer pair: state in → string out for rendering, and
// UpdateModelPicker for keyboard navigation. Neither function allocates
// goroutines or reads wall clocks.
//
// Provider column layout matches the upstream 2-per-row grid. The model
// column appears only after a provider is selected and shows a scrolling
// list with the current model marked by "*".
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// ProviderEntry is one provider option in the picker.
type ProviderEntry struct {
	ID    string
	Label string
}

// ModelEntry is one model option shown after a provider is selected.
type ModelEntry struct {
	ID    string
	Label string
}

// ModelPickerState is the complete state for the model picker overlay.
// Width and Height carry the terminal dimensions for layout calculations.
type ModelPickerState struct {
	Width    int
	Height   int

	// Providers is the list of available providers.
	Providers []ProviderEntry

	// SelectedProviderIndex is the 0-based index into Providers that is
	// currently focused by the user. -1 means no provider selected yet.
	SelectedProviderIndex int

	// Models is the list of models for the selected provider. Populated
	// by the caller after the provider is selected.
	Models []ModelEntry

	// SelectedModelIndex is the 0-based index into Models that is currently
	// focused. -1 means no model selected yet.
	SelectedModelIndex int

	// CurrentProvider is the provider ID of the currently active model.
	// It is used to mark the current model with "*" in the model list.
	CurrentProvider string

	// CurrentModel is the currently active model ID. It is marked with "*".
	CurrentModel string

	// Confirming is true when the user has pressed Enter on a model and
	// the picker should emit the confirmed selection.
	Confirming bool
}

// ModelPickerResult is returned when the user confirms a model selection.
// It carries the chosen provider and model IDs.
type ModelPickerResult struct {
	Provider string
	Model    string
}

// modelPickerConfirmedMsg is the internal Bubble Tea message emitted when
// the user confirms a model selection.
type modelPickerConfirmedMsg ModelPickerResult

// RenderModelPicker renders the model picker overlay as a string.
// It shows a 2-column provider grid; once a provider is selected it also
// shows a model list column to the right. The current model is marked with
// "*". Keyboard hints render at the bottom.
//
// Layout:
//   - Title: "Select Model"
//   - Provider grid: 2 entries per row, selected provider highlighted with "❯"
//   - Model column: appears when provider selected, scrolling list with "*" for current
//   - Footer: keyboard hints (↑/↓ navigate, Enter confirm, Esc cancel)
func RenderModelPicker(state ModelPickerState) string {
	if state.Width < 30 {
		return renderModelPickerNarrow(state)
	}
	return renderModelPickerWide(state)
}

func renderModelPickerWide(state ModelPickerState) string {
	var b strings.Builder

	// Title
	title := "  Select Model  "
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("39")).
		Bold(true).
		Width(state.Width).
		Render(title))
	b.WriteString("\n\n")

	// Provider section header (only show when providers exist)
	if len(state.Providers) > 0 {
		b.WriteString("  Provider\n")
	}

	// Provider grid: 2 per row
	providerCols := 2
	providerColWidth := (state.Width - 4) / providerCols
	if providerColWidth < 20 {
		providerColWidth = 20
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")). // bright cyan
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // muted white

	for row := 0; row < len(state.Providers); row += providerCols {
		line := "  "
		for col := 0; col < providerCols && row+col < len(state.Providers); col++ {
			idx := row + col
			entry := state.Providers[idx]
			prefix := "  "
			style := normalStyle

			if idx == state.SelectedProviderIndex {
				prefix = "❯ "
				style = selectedStyle
			}

			// Truncate label if needed
			label := entry.Label
			if len(label) > providerColWidth-3 {
				label = label[:providerColWidth-6] + "…"
			}

			padded := label + strings.Repeat(" ", providerColWidth-len(label)-3)
			line += prefix + style.Render(padded)
		}
		b.WriteString(line + "\n")
	}

	// Model column when provider is selected
	if state.SelectedProviderIndex >= 0 && state.SelectedProviderIndex < len(state.Providers) {
		b.WriteString("\n")
		selectedProv := state.Providers[state.SelectedProviderIndex]
		b.WriteString("  Model (" + selectedProv.Label + ")\n")

		// Calculate model list height
		modelStartY := 4 + (len(state.Providers)+1)/2 // title + header + provider rows
		modelMaxHeight := state.Height - modelStartY - 3 // -3 for hints

		modelSelectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")).
			Bold(true)
		modelNormalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
		currentModelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")) // green for current

		visibleModels := state.Models
		if modelMaxHeight > 0 && len(visibleModels) > modelMaxHeight {
			// Scroll window: center on selected if possible
			half := modelMaxHeight / 2
			start := state.SelectedModelIndex - half
			if start < 0 {
				start = 0
			}
			end := start + modelMaxHeight
			if end > len(visibleModels) {
				end = len(visibleModels)
				start = end - modelMaxHeight
				if start < 0 {
					start = 0
				}
			}
			visibleModels = visibleModels[start:end]
		}

		for _, entry := range visibleModels {
			prefix := "  "
			style := modelNormalStyle
			marker := "  "

			// Check if this is the selected model
			if state.SelectedModelIndex >= 0 && state.SelectedModelIndex < len(state.Models) {
				if state.Models[state.SelectedModelIndex].ID == entry.ID {
					prefix = "❯ "
					style = modelSelectedStyle
				}
			}

			// Mark current model with "*"
			isCurrent := entry.ID == state.CurrentModel && selectedProv.ID == state.CurrentProvider
			if isCurrent {
				marker = "* "
				style = currentModelStyle
			}

			// Truncate model label
			modelLabel := entry.Label
			modelColWidth := state.Width - 6
			if len(modelLabel) > modelColWidth-4 {
				modelLabel = modelLabel[:modelColWidth-7] + "…"
			}

			b.WriteString(prefix + marker + style.Render(modelLabel) + "\n")
		}
	}

	// Keyboard hints at bottom
	b.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	b.WriteString(hintStyle.Render("  ↑/↓ navigate  ·  Enter confirm  ·  Esc cancel"))

	return b.String()
}

func renderModelPickerNarrow(state ModelPickerState) string {
	var b strings.Builder

	title := "  Select Model  "
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("39")).
		Bold(true).
		Width(state.Width).
		Render(title))
	b.WriteString("\n\n")

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Single column providers
	for i, entry := range state.Providers {
		prefix := "  "
		style := normalStyle
		if i == state.SelectedProviderIndex {
			prefix = "❯ "
			style = selectedStyle
		}
		b.WriteString(prefix + style.Render(entry.Label) + "\n")
	}

	// Model list when provider selected
	if state.SelectedProviderIndex >= 0 && state.SelectedProviderIndex < len(state.Providers) {
		b.WriteString("\n")
		selectedProv := state.Providers[state.SelectedProviderIndex]
		b.WriteString("  Models for " + selectedProv.Label + ":\n")

		modelSelectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")).
			Bold(true)
		modelNormalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
		currentModelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

		for _, entry := range state.Models {
			prefix := "  "
			style := modelNormalStyle
			marker := "  "

			if state.SelectedModelIndex >= 0 && state.SelectedModelIndex < len(state.Models) {
				if state.Models[state.SelectedModelIndex].ID == entry.ID {
					prefix = "❯ "
					style = modelSelectedStyle
				}
			}

			isCurrent := entry.ID == state.CurrentModel && selectedProv.ID == state.CurrentProvider
			if isCurrent {
				marker = "* "
				style = currentModelStyle
			}

			b.WriteString(prefix + marker + style.Render(entry.Label) + "\n")
		}
	}

	b.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	b.WriteString(hintStyle.Render("  ↑/↓ navigate  ·  Enter confirm  ·  Esc cancel"))

	return b.String()
}

// UpdateModelPicker handles keyboard events for the model picker overlay.
// It returns the updated state and an optional Bubble Tea command to execute.
func UpdateModelPicker(msg tea.Msg, state ModelPickerState) (ModelPickerState, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return handleModelPickerKey(msg, state)
	}
	return state, nil
}

func handleModelPickerKey(msg tea.KeyMsg, state ModelPickerState) (ModelPickerState, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if state.SelectedModelIndex >= 0 {
			// Navigate up in model list
			if state.SelectedModelIndex > 0 {
				state.SelectedModelIndex--
			}
		} else {
			// Navigate up in provider list
			if state.SelectedProviderIndex > 0 {
				state.SelectedProviderIndex--
			}
		}
		return state, nil

	case tea.KeyDown:
		if state.SelectedModelIndex >= 0 {
			// Navigate down in model list
			if state.SelectedModelIndex < len(state.Models)-1 {
				state.SelectedModelIndex++
			}
		} else {
			// Navigate down in provider list
			if state.SelectedProviderIndex < len(state.Providers)-1 {
				state.SelectedProviderIndex++
			}
		}
		return state, nil

	case tea.KeyLeft:
		// Left switches from model list back to provider list
		if state.SelectedModelIndex >= 0 {
			state.SelectedModelIndex = -1
		}
		return state, nil

	case tea.KeyRight:
		// Right switches from provider list to model list
		if state.SelectedProviderIndex >= 0 && state.SelectedModelIndex < 0 {
			// Select first model when entering model list
			if len(state.Models) > 0 {
				state.SelectedModelIndex = 0
			}
		}
		return state, nil

	case tea.KeyEnter:
		// Confirm selection
		if state.SelectedProviderIndex >= 0 {
			selectedProv := state.Providers[state.SelectedProviderIndex]
			var selectedModel string

			if state.SelectedModelIndex >= 0 && state.SelectedModelIndex < len(state.Models) {
				selectedModel = state.Models[state.SelectedModelIndex].ID
			} else {
				// No model selected yet, use current
				selectedModel = state.CurrentModel
			}

			return state, func() tea.Msg {
				return modelPickerConfirmedMsg{
					Provider: selectedProv.ID,
					Model:    selectedModel,
				}
			}
		}
		return state, nil

	case tea.KeyEscape:
		// Cancel - return empty result
		return state, func() tea.Msg {
			return modelPickerConfirmedMsg{}
		}
	}
	return state, nil
}
