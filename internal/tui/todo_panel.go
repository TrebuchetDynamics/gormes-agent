// Package tui — Hermes-compatible TodoPanel renderer.
//
// This module provides a collapsible todo list renderer that mirrors the
// behavior of upstream Hermes' todoPanel.tsx component.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TodoStatus represents the completion state of a todo item.
type TodoStatus int

const (
	// TodoStatusPending indicates an incomplete task.
	TodoStatusPending TodoStatus = iota
	// TodoStatusDone indicates a completed task.
	TodoStatusDone
)

// Glyph returns the Unicode glyph used to render this status.
func (s TodoStatus) Glyph() string {
	switch s {
	case TodoStatusPending:
		return "○"
	case TodoStatusDone:
		return "●"
	default:
		return "○"
	}
}

// TodoItem represents a single task in the todo panel.
type TodoItem struct {
	Text      string
	Status    TodoStatus
	Collapsed bool
}

// RenderTodoPanel renders a collapsible todo list for the given items.
// It returns an empty string if items is nil or empty.
// Width constrains the panel; items may be truncated to fit.
func RenderTodoPanel(items []TodoItem, width int) string {
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
			text = trimToWidth(text, maxTextWidth)
		}

		var line string
		if item.Collapsed {
			line = glyph + " ▸ " + text
		} else {
			line = glyph + "   " + text
		}

		// Add done styling (strikethrough-like dimming) via line prefix
		if item.Status == TodoStatusDone {
			lines = append(lines, dimLine(line))
		} else {
			lines = append(lines, line)
		}
	}

	// If width is too narrow, return compact single line
	if width > 0 && width < 20 {
		return strings.TrimSpace(compactTodoSummary(items, width))
	}

	return strings.Join(lines, "\n")
}

// trimToWidth trims text to fit within maxWidth using lipgloss width.
func trimToWidth(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	const ellipsis = "…"
	ellipsisWidth := lipgloss.Width(ellipsis)
	if maxWidth <= ellipsisWidth {
		return strings.Repeat(".", maxWidth)
	}

	var b strings.Builder
	used := 0
	for _, r := range text {
		rw := lipgloss.Width(string(r))
		if used+rw+ellipsisWidth > maxWidth {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + ellipsis
}

// dimLine returns a visually dimmed version of the line for done items.
func dimLine(line string) string {
	// Use dim style prefix to indicate completed task
	return "  " + line
}

// compactTodoSummary produces a one-line summary when width is very narrow.
func compactTodoSummary(items []TodoItem, width int) string {
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
		summary.WriteString(string(rune('0' + len(items)-1)))
	}
	result := summary.String()
	if lipgloss.Width(result) > width {
		return trimToWidth(result, width)
	}
	return result
}
