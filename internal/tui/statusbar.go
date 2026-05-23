package tui

import "strings"

type StatusBarMode string

const (
	StatusBarModeTop    StatusBarMode = "top"
	StatusBarModeBottom StatusBarMode = "bottom"
	StatusBarModeOff    StatusBarMode = "off"
)

const statusBarSlashUsage = "usage: /statusbar [on|off|top|bottom|toggle]"

func normalizeStatusBarMode(mode StatusBarMode) StatusBarMode {
	switch mode {
	case StatusBarModeTop, StatusBarModeBottom, StatusBarModeOff:
		return mode
	default:
		return StatusBarModeTop
	}
}

func statusbarSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "statusbar: TUI unavailable"}
	}
	next, ok := statusBarSlashNext(input, model.statusBarMode)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: statusBarSlashUsage}
	}
	model.statusBarMode = next
	return SlashResult{Handled: true, StatusMessage: "status bar " + string(next)}
}

func statusBarSlashNext(input string, current StatusBarMode) (StatusBarMode, bool) {
	current = normalizeStatusBarMode(current)
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return toggledStatusBarMode(current), true
	}
	if len(fields) > 2 {
		return current, false
	}
	switch strings.ToLower(fields[1]) {
	case "on", "top":
		return StatusBarModeTop, true
	case "bottom":
		return StatusBarModeBottom, true
	case "off":
		return StatusBarModeOff, true
	case "toggle":
		return toggledStatusBarMode(current), true
	default:
		return current, false
	}
}

func toggledStatusBarMode(current StatusBarMode) StatusBarMode {
	if normalizeStatusBarMode(current) == StatusBarModeOff {
		return StatusBarModeTop
	}
	return StatusBarModeOff
}
