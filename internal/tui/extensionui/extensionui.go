package extensionui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// Text is one session-scoped extension status entry.
type Text struct {
	SessionID string
	Text      string
}

// LinesState is one session-scoped extension line block.
type LinesState struct {
	SessionID string
	Lines     []string
}

// Widget is one session-scoped extension widget block.
type Widget struct {
	SessionID string
	Lines     []string
	Placement kernel.ExtensionUIWidgetPlacement
}

// Working is one session-scoped extension working-indicator override.
type Working struct {
	SessionID     string
	Text          string
	Frames        []string
	HideIndicator bool
}

// Key normalizes an extension UI key without imposing product policy.
func Key(key string) string {
	return strings.TrimSpace(key)
}

// Lines trims extension-provided line content and drops blank rows.
func Lines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r\n")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// StatusEntries returns deterministic status text for a session.
func StatusEntries(statuses map[string]Text, sessionID string) []string {
	if len(statuses) == 0 {
		return nil
	}
	keys := make([]string, 0, len(statuses))
	for key, entry := range statuses {
		if entry.SessionID == sessionID {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		text := strings.TrimSpace(statuses[key].Text)
		if text != "" {
			entries = append(entries, text)
		}
	}
	return entries
}

// StatusLine appends extension status entries to a base status row within width.
func StatusLine(base string, entries []string, width int, trim func(string, int) string) string {
	if len(entries) == 0 {
		return trim(base, width)
	}
	suffix := " │ " + strings.Join(entries, " │ ")
	suffix = trim(suffix, width)
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(suffix) >= width {
		return suffix
	}
	baseWidth := width - lipgloss.Width(suffix)
	return trim(base, baseWidth) + suffix
}

// RenderLines renders extension-provided lines within the supplied width.
func RenderLines(lines []string, width int, trim func(string, int) string) string {
	if width <= 0 {
		return ""
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, trim(line, width))
	}
	return strings.Join(out, "\n")
}

// WidgetBlocks returns deterministic widget blocks for a session and placement.
func WidgetBlocks(widgets map[string]Widget, sessionID string, placement kernel.ExtensionUIWidgetPlacement, width int, trim func(string, int) string) string {
	placement = kernel.NormalizeExtensionUIWidgetPlacement(placement)
	if len(widgets) == 0 {
		return ""
	}
	keys := make([]string, 0, len(widgets))
	for key, widget := range widgets {
		if widget.SessionID == sessionID && kernel.NormalizeExtensionUIWidgetPlacement(widget.Placement) == placement {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	blocks := make([]string, 0, len(keys))
	for _, key := range keys {
		blocks = append(blocks, RenderLines(widgets[key].Lines, width, trim))
	}
	return strings.Join(blocks, "\n")
}
