package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"

type StatusBarMode = statusbar.Mode

const (
	StatusBarModeTop    = statusbar.ModeTop
	StatusBarModeBottom = statusbar.ModeBottom
	StatusBarModeOff    = statusbar.ModeOff
)

const statusBarSlashUsage = statusbar.SlashUsage

func normalizeStatusBarMode(mode StatusBarMode) StatusBarMode {
	return statusbar.NormalizeMode(mode)
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
	return statusbar.SlashNext(input, current)
}

func toggledStatusBarMode(current StatusBarMode) StatusBarMode {
	return statusbar.ToggledMode(current)
}
