package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/skills"
)

// RenderSkillsPane renders the shared skills browse view for the TUI pane.
// The text surface intentionally matches the gateway /skills output so
// operators see the same listing on both edges; the lipgloss wrapping is
// purely cosmetic (width-aware line wrap, no color tokens stripped).
func RenderSkillsPane(view skills.BrowseView, width int) string {
	raw := skills.FormatBrowseSummary(view)
	if width <= 0 {
		return raw
	}
	wrap := lipgloss.NewStyle()
	if width > 4 {
		wrap = wrap.Width(width - 2)
	}
	var wrapped []string
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, wrap.Render(line))
	}
	return strings.Join(wrapped, "\n")
}
