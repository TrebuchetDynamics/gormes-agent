package transientpage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// State is a local, dismissible page rendered above the composer.
type State struct {
	Title  string
	Body   string
	Offset int
}

// Styles carries the skin-derived styles used by the page chrome.
type Styles struct {
	Title     lipgloss.Style
	Dim       lipgloss.Style
	Separator lipgloss.Style
	Text      lipgloss.Style
}

// WrapFunc wraps and trims one line to the requested display width.
type WrapFunc func(string, int) string

// Render renders the transient page chrome. The caller supplies styling and
// markdown wrapping so this package stays independent of the root TUI model.
func Render(page State, width, height int, styles Styles, styled bool, wrap WrapFunc) string {
	if wrap == nil {
		wrap = func(line string, _ int) string { return line }
	}
	title := strings.TrimSpace(page.Title)
	if title == "" {
		title = "Page"
	}
	bodyWidth := width - 4
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	lines := Lines(page.Body, bodyWidth, wrap)
	if page.Offset > 0 && page.Offset < len(lines) {
		lines = lines[page.Offset:]
	}
	if height > 0 && len(lines) > height {
		omitted := len(lines) - height
		if height <= 1 {
			lines = []string{fmt.Sprintf("... %d more lines ...", omitted+1)}
		} else {
			lines = append(lines[:height-1], fmt.Sprintf("... %d more lines ...", omitted+1))
		}
	}

	out := make([]string, 0, len(lines)+2)
	out = append(out, renderStyle(styled, styles.Title, "╭─ "+title))
	if len(lines) == 0 {
		out = append(out, renderStyle(styled, styles.Dim, "│ (empty)"))
	} else {
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				out = append(out, renderStyle(styled, styles.Separator, "│"))
				continue
			}
			for _, wrapped := range strings.Split(wrap(line, bodyWidth), "\n") {
				out = append(out, renderStyle(styled, styles.Separator, "│ ")+renderStyle(styled, styles.Text, wrapped))
			}
		}
	}
	out = append(out, renderStyle(styled, styles.Dim, "╰─ Esc to close"))
	return strings.Join(out, "\n")
}

// Lines normalizes page body text into wrapped display lines.
func Lines(body string, width int, wrap WrapFunc) []string {
	if wrap == nil {
		wrap = func(line string, _ int) string { return line }
	}
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return nil
	}
	raw := strings.Split(body, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if width > 0 {
			line = wrap(line, width)
		}
		parts := strings.Split(line, "\n")
		lines = append(lines, parts...)
	}
	return lines
}

func renderStyle(enabled bool, style lipgloss.Style, value string) string {
	if !enabled {
		return value
	}
	return style.Render(value)
}
