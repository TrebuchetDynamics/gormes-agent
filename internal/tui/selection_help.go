package tui

// TerminalNativeSelectionHelp documents Gormes' selection-and-copy stance:
// the native Bubble Tea TUI relies on the host terminal's own selection
// mechanic and does not reproduce the upstream ui-tui custom selection-copy
// keybindings. A future Go-native copy mode, if it ships, must replace this
// constant rather than extend it; until then the wording deliberately
// promises no in-app shortcut so users do not chase a feature that does
// not exist.
const TerminalNativeSelectionHelp = "Selection: use your terminal's native selection (Shift-drag in most terminals; iTerm Cmd-drag, tmux copy-mode). Gormes does not advertise an in-app copy hotkey."

// SelectionHelpLine returns TerminalNativeSelectionHelp. Callers that want
// the help string should route through this helper so a future Go-native
// copy-mode helper can swap the implementation in one place.
func SelectionHelpLine() string {
	return TerminalNativeSelectionHelp
}
