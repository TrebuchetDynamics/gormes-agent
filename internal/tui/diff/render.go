package diff

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles carries the skin-derived styles needed to render unified diff lines.
type Styles struct {
	Minus lipgloss.Style
	Plus  lipgloss.Style
	Hunk  lipgloss.Style
	File  lipgloss.Style
}

// Render colors a unified diff using caller-supplied styles. The renderer is
// pure: it owns line classification and max-line truncation, while package tui
// keeps skin lookup and the public RenderDiff compatibility seam.
func Render(styles Styles, diffText string, maxLines int) string {
	lines := strings.Split(diffText, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	var b strings.Builder
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			b.WriteString(styles.File.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(styles.Hunk.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(styles.Minus.Render(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(styles.Plus.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
