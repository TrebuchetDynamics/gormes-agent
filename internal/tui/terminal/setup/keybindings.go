package setup

import keybindingpolicy "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/keybindings"

type terminalKeybindingAnalysis = keybindingpolicy.TerminalAnalysis

func analyzeTerminalKeybindings(existing []map[string]any, platform string) terminalKeybindingAnalysis {
	return keybindingpolicy.AnalyzeTerminalKeybindings(existing, platform)
}
