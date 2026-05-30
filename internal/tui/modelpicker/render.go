package modelpicker

import (
	"strings"

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
func Render(state State, styles Styles) string {
	if state.Width < 30 {
		return renderNarrow(state, styles)
	}
	return renderWide(state, styles)
}

func renderWide(state State, styles Styles) string {
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
		b.WriteString(styles.Label.Render("  Model ("+selectedProv.Label+")") + "\n")

		// Calculate model list height
		modelStartY := 4 + (len(state.Providers)+1)/2    // title + header + provider rows
		modelMaxHeight := state.Height - modelStartY - 3 // -3 for hints

		modelSelectedStyle := styles.Selected
		modelNormalStyle := styles.Normal
		currentModelStyle := styles.Good

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
	b.WriteString(styles.Dim.Render("  ↑/↓ navigate  ·  Enter confirm  ·  Esc cancel"))

	return b.String()
}

func renderNarrow(state State, styles Styles) string {
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
	if state.SelectedProviderIndex >= 0 && state.SelectedProviderIndex < len(state.Providers) {
		b.WriteString("\n")
		selectedProv := state.Providers[state.SelectedProviderIndex]
		b.WriteString(styles.Label.Render("  Models for "+selectedProv.Label+":") + "\n")

		modelSelectedStyle := styles.Selected
		modelNormalStyle := styles.Normal
		currentModelStyle := styles.Good

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
	b.WriteString(styles.Dim.Render("  ↑/↓ navigate  ·  Enter confirm  ·  Esc cancel"))

	return b.String()
}
