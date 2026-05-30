package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/help"

// helpSlashHandler returns a compact in-TUI command inventory. Hermes ui-tui
// renders /help as a local panel assembled from its slash registry; the native
// Bubble Tea TUI does not yet have a general transcript panel for local slash
// handlers, so this first port keeps the behavior local and visible via the
// existing status row instead of falling through to the unavailable fallback.
func helpSlashHandler(_ string, _ *Model) SlashResult {
	return SlashResult{Handled: true, StatusMessage: nativeTUIHelpStatus()}
}

func nativeTUIHelpStatus() string {
	return help.NativeStatus()
}
