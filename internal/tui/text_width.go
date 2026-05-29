package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"

// trimToWidth trims text to fit within maxWidth using lipgloss width.
func trimToWidth(text string, maxWidth int) string {
	return ansitext.TrimToWidth(text, maxWidth)
}
