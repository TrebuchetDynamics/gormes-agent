package view

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract"

	"github.com/charmbracelet/lipgloss"
)

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
//
// Styles contains the semantic Lip Gloss styles needed by the model picker renderer.
type Styles struct {
	ActivePill lipgloss.Style
	Label      lipgloss.Style
	Selected   lipgloss.Style
	Normal     lipgloss.Style
	Good       lipgloss.Style
	Dim        lipgloss.Style
}

// Render renders the model picker overlay as a string.
func Render(state contract.State, styles Styles) string {
	if state.Width < 30 {
		return renderNarrow(state, styles)
	}
	return renderWide(state, styles)
}

func renderWide(state contract.State, styles Styles) string {
	var b strings.Builder

	// Title
	title := "  Select Model  "
	b.WriteString(styles.ActivePill.Width(state.Width).Render(title))
	b.WriteString("\n\n")

	// Provider section header (only show when providers exist)
	if len(state.Providers) > 0 {
		b.WriteString(styles.Label.Render("  Provider") + "\n")
	}

	// Provider grid: 2 per row
	providerCols := 2
	providerColWidth := (state.Width - 4) / providerCols
	if providerColWidth < 20 {
		providerColWidth = 20
	}

	selectedStyle := styles.Selected
	normalStyle := styles.Normal

	for row := 0; row < len(state.Providers); row += providerCols {
		line := "  "
		for col := 0; col < providerCols && row+col < len(state.Providers); col++ {
			idx := row + col
			line += providerRow(state.Providers[idx], idx == state.SelectedProviderIndex, providerColWidth, selectedStyle, normalStyle)
		}
		b.WriteString(line + "\n")
	}

	// Model column when provider is selected
	if selectedProv, ok := selectedProvider(state); ok {
		b.WriteString("\n")
		b.WriteString(styles.Label.Render("  Model ("+selectedProv.Label+")") + "\n")

		// Calculate model list height
		modelStartY := 4 + (len(state.Providers)+1)/2    // title + header + provider rows
		modelMaxHeight := state.Height - modelStartY - 3 // -3 for hints

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
			b.WriteString(renderModelRow(modelRow{
				entry:    entry,
				provider: selectedProv,
				state:    state,
				maxWidth: state.Width - 6,
				truncate: true,
				styles:   styles,
			}) + "\n")
		}
	}

	// Keyboard hints at bottom
	b.WriteString("\n")
	b.WriteString(styles.Dim.Render("  ↑/↓ navigate  ·  Enter confirm  ·  Esc cancel"))

	return b.String()
}

func renderNarrow(state contract.State, styles Styles) string {
	var b strings.Builder

	title := "  Select Model  "
	b.WriteString(styles.ActivePill.Width(state.Width).Render(title))
	b.WriteString("\n\n")

	selectedStyle := styles.Selected
	normalStyle := styles.Normal

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
	if selectedProv, ok := selectedProvider(state); ok {
		b.WriteString("\n")
		b.WriteString(styles.Label.Render("  Models for "+selectedProv.Label+":") + "\n")

		for _, entry := range state.Models {
			b.WriteString(renderModelRow(modelRow{
				entry:    entry,
				provider: selectedProv,
				state:    state,
				styles:   styles,
			}) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.Dim.Render("  ↑/↓ navigate  ·  Enter confirm  ·  Esc cancel"))

	return b.String()
}
