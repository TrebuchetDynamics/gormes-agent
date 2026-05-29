// Package todo renders Hermes-compatible todo panels for terminal UIs.
package todo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"
)

// Status represents the completion state of a todo item.
type Status int

const (
	// StatusPending indicates an incomplete task.
	StatusPending Status = iota
	// StatusDone indicates a completed task.
	StatusDone
)

// Glyph returns the Unicode glyph used to render this status.
func (s Status) Glyph() string {
	switch s {
	case StatusPending:
		return "○"
	case StatusDone:
		return "●"
	default:
		return "○"
	}
}

// Item represents a single task in the todo panel.
type Item struct {
	Text      string
	Status    Status
	Collapsed bool
}

// Styles supplies caller-owned styling functions for the todo panel.
type Styles struct {
	Accent func(string) string
	Good   func(string) string
	Dim    func(string) string
	Text   func(string) string
}

// Render renders a collapsible todo list for the given items.
// It returns an empty string if items is nil or empty.
// Width constrains the panel; items may be truncated to fit.
func Render(items []Item, width int) string {
	return renderWithStyles(items, width, Styles{}, false)
}

// RenderWithStyles renders a todo panel with caller-supplied styling.
func RenderWithStyles(items []Item, width int, styles Styles) string {
	return renderWithStyles(items, width, styles, true)
}

func renderWithStyles(items []Item, width int, styles Styles, styled bool) string {
	if len(items) == 0 {
		return ""
	}

	// Determine if we need collapse indicators
	hasCollapsed := false
	for _, item := range items {
		if item.Collapsed {
			hasCollapsed = true
			break
		}
	}

	// If width is too narrow, return compact single line
	if width > 0 && width < 20 {
		return renderStyle(styled, styles.Dim, strings.TrimSpace(compactSummary(items, width)))
	}

	// Build lines
	var lines []string
	for _, item := range items {
		glyph := item.Status.Glyph()
		text := strings.TrimSpace(item.Text)

		// Truncate text if needed to fit width
		maxTextWidth := width - 4 // space for glyph + spaces
		if hasCollapsed {
			maxTextWidth -= 2 // space for collapse indicator
		}
		if maxTextWidth < 1 {
			maxTextWidth = 1
		}
		if lipgloss.Width(text) > maxTextWidth {
			text = ansitext.TrimToWidth(text, maxTextWidth)
		}

		if styled {
			glyphStyle := styles.Accent
			textStyle := styles.Text
			if item.Status == StatusDone {
				glyphStyle = styles.Good
				textStyle = styles.Dim
			}
			marker := "   "
			if item.Collapsed {
				marker = " " + renderStyle(true, styles.Dim, "▸") + " "
			}
			lines = append(lines, renderStyle(true, glyphStyle, glyph)+marker+renderStyle(true, textStyle, text))
			continue
		}

		var line string
		if item.Collapsed {
			line = glyph + " ▸ " + text
		} else {
			line = glyph + "   " + text
		}

		// Add done styling (strikethrough-like dimming) via line prefix
		if item.Status == StatusDone {
			lines = append(lines, dimLine(line))
		} else {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// dimLine returns a visually dimmed version of the line for done items.
func dimLine(line string) string {
	// Use dim style prefix to indicate completed task
	return "  " + line
}

// compactSummary produces a one-line summary when width is very narrow.
func compactSummary(items []Item, width int) string {
	if len(items) == 0 {
		return ""
	}
	summary := strings.Builder{}
	for _, item := range items {
		summary.WriteString(item.Status.Glyph())
	}
	summary.WriteString(" ")
	summary.WriteString(strings.TrimSpace(items[0].Text))
	if len(items) > 1 {
		summary.WriteString(" +")
		summary.WriteString(string(rune('0' + len(items) - 1)))
	}
	result := summary.String()
	if lipgloss.Width(result) > width {
		return ansitext.TrimToWidth(result, width)
	}
	return result
}

func renderStyle(styled bool, style func(string) string, text string) string {
	if !styled || style == nil {
		return text
	}
	return style(text)
}
