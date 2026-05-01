package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderDiff(skin HermesSkin, diffText string, maxLines int) string {
	lines := strings.Split(diffText, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	minusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#601010"))
	plusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#106010"))
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(skin.Colors.SessionBorder))
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(skin.Colors.SessionLabel))

	var b strings.Builder
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			b.WriteString(fileStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(hunkStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(minusStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(plusStyle.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
