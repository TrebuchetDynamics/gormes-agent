package view

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract"
	"github.com/charmbracelet/lipgloss"
)

type modelRow struct {
	entry    contract.ModelEntry
	provider contract.ProviderEntry
	state    contract.State
	maxWidth int
	truncate bool
	styles   Styles
}

func selectedProvider(state contract.State) (contract.ProviderEntry, bool) {
	if state.SelectedProviderIndex < 0 || state.SelectedProviderIndex >= len(state.Providers) {
		return contract.ProviderEntry{}, false
	}
	return state.Providers[state.SelectedProviderIndex], true
}

func selectedModelID(state contract.State) (string, bool) {
	if state.SelectedModelIndex < 0 || state.SelectedModelIndex >= len(state.Models) {
		return "", false
	}
	return state.Models[state.SelectedModelIndex].ID, true
}

func renderModelRow(row modelRow) string {
	prefix := "  "
	style := row.styles.Normal
	marker := "  "

	if selectedID, ok := selectedModelID(row.state); ok && selectedID == row.entry.ID {
		prefix = "❯ "
		style = row.styles.Selected
	}

	if row.entry.ID == row.state.CurrentModel && row.provider.ID == row.state.CurrentProvider {
		marker = "* "
		style = row.styles.Good
	}

	label := row.entry.Label
	if row.truncate {
		label = truncateLabel(label, row.maxWidth, 4)
	}
	return prefix + marker + style.Render(label)
}

func providerRow(entry contract.ProviderEntry, selected bool, width int, selectedStyle, normalStyle lipgloss.Style) string {
	prefix := "  "
	style := normalStyle
	if selected {
		prefix = "❯ "
		style = selectedStyle
	}
	label := truncateLabel(entry.Label, width, 3)
	padded := label + strings.Repeat(" ", width-len(label)-3)
	return prefix + style.Render(padded)
}

func truncateLabel(label string, width int, markerSpace int) string {
	if len(label) > width-markerSpace {
		return label[:width-markerSpace-3] + "…"
	}
	return label
}
