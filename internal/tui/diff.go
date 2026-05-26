package tui

import "strings"

func RenderDiff(skin HermesSkin, diffText string, maxLines int) string {
	lines := strings.Split(diffText, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	styles := SkinStylesFor(skin)
	minusStyle := styles.Bad
	plusStyle := styles.Good
	hunkStyle := styles.Separator
	fileStyle := styles.Label

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
