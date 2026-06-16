package layout

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// InputWidth returns the Bubble component content width that fits inside the
// terminal while preserving room for prompt/cursor cells.
func InputWidth(terminalWidth int) int {
	if terminalWidth <= 0 {
		return 76
	}
	return Max(1, terminalWidth-4)
}

// WrapIndented wraps text to width with prefix on the first line and spaces of
// equal display width on continuation lines.
func WrapIndented(prefix, text string, width int) string {
	if width <= 0 {
		width = 80
	}
	if prefix == "" {
		return WrapText(text, width)
	}
	indentWidth := lipgloss.Width(prefix)
	bodyWidth := Max(1, width-indentWidth)
	wrapped := WrapText(text, bodyWidth)
	lines := strings.Split(wrapped, "\n")
	indent := strings.Repeat(" ", indentWidth)
	for i, line := range lines {
		if i == 0 {
			lines[i] = prefix + line
			continue
		}
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// WrapText wraps text to width, preserving explicit line breaks.
func WrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		out = append(out, wrapLine(line, width)...)
	}
	return strings.Join(out, "\n")
}

func wrapLine(line string, width int) []string {
	line = strings.TrimRight(line, " \t")
	if line == "" {
		return []string{""}
	}
	var lines []string
	for lipgloss.Width(line) > width {
		cut := wrapCut(line, width)
		lines = append(lines, strings.TrimRight(line[:cut], " \t"))
		line = strings.TrimLeft(line[cut:], " \t")
		if line == "" {
			break
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func wrapCut(line string, width int) int {
	if width <= 0 {
		_, size := utf8.DecodeRuneInString(line)
		return Max(1, size)
	}
	lastSpace := -1
	used := 0
	for i, r := range line {
		if r == ' ' || r == '\t' {
			lastSpace = i
		}
		rw := lipgloss.Width(string(r))
		if used+rw > width {
			if lastSpace > 0 {
				return lastSpace
			}
			if i > 0 {
				return i
			}
			return i + len(string(r))
		}
		used += rw
	}
	return len(line)
}

// TrimTrailingWhitespace removes trailing spaces and tabs from every line.
func TrimTrailingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// ClampView keeps the most useful setup wizard lines visible on very short
// terminals.
func ClampView(s string, width, height int) string {
	if height <= 0 || height > 10 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= height {
		return s
	}
	if height <= 2 {
		return TrimToWidth("terminal too small; resize", width)
	}
	marker := TrimToWidth("… omitted; resize", width)
	out := make([]string, 0, height)
	out = append(out, lines[0])
	if height > 5 {
		if prompt := promptLine(lines); prompt != "" {
			out = append(out, TrimToWidth(prompt, width))
		} else if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
			out = append(out, lines[1])
		}
	}
	out = append(out, marker)
	if focal := focalLine(lines); focal != "" && len(out) < height-2 {
		out = append(out, TrimToWidth(focal, width))
	}
	for _, line := range helpTail(lines, height-len(out)) {
		out = append(out, line)
	}
	for len(out) > height {
		out = append(out[:len(out)-2], out[len(out)-1])
	}
	return strings.Join(out, "\n")
}

func promptLine(lines []string) string {
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "→") || strings.Contains(trimmed, "submit") || strings.Contains(trimmed, "abort") {
			continue
		}
		return line
	}
	return ""
}

func focalLine(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "→") || strings.HasPrefix(trimmed, "Filter:") {
			return line
		}
	}
	return ""
}

func helpTail(lines []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	start := len(lines) - limit
	if start < 0 {
		start = 0
	}
	return append([]string(nil), lines[start:]...)
}

// TrimToWidth shortens text to display width and appends an ellipsis when it
// does not fit.
func TrimToWidth(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	ellipsis := "…"
	limit := width - lipgloss.Width(ellipsis)
	used := 0
	for i, r := range text {
		rw := lipgloss.Width(string(r))
		if used+rw > limit {
			return strings.TrimRight(text[:i], " \t") + ellipsis
		}
		used += rw
	}
	return text
}

// Max returns the larger integer.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
