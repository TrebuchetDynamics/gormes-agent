package setup

import keybindingpolicy "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/keybindings"

type keybindingMatchState = keybindingpolicy.MatchState

const (
	keybindingMissing    = keybindingpolicy.Missing
	keybindingEquivalent = keybindingpolicy.Equivalent
	keybindingConflict   = keybindingpolicy.Conflict
)

func keybindingState(existing []map[string]any, desired map[string]any) (keybindingMatchState, string) {
	return keybindingpolicy.State(existing, desired)
}

func defaultTerminalKeybindings(platform string) []map[string]any {
	return keybindingpolicy.DefaultTerminalKeybindings(platform)
}
