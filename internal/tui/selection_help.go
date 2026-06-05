package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"

// TerminalNativeSelectionHelp documents Gormes' selection-and-copy stance at
// the root TUI compatibility seam.
const TerminalNativeSelectionHelp = terminal.NativeSelectionHelp

// SelectionHelpLine returns TerminalNativeSelectionHelp. Callers that want the
// help string should route through this helper so a future Go-native copy-mode
// helper can swap the implementation in one place.
func SelectionHelpLine() string {
	return terminal.SelectionHelpLine()
}
