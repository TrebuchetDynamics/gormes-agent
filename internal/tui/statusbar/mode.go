package statusbar

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar/modes"

// Mode controls where the Hermes-compatible status rule renders.
type Mode = modes.Mode

const (
	ModeTop    Mode = modes.ModeTop
	ModeBottom Mode = modes.ModeBottom
	ModeOff    Mode = modes.ModeOff
)

const SlashUsage = modes.SlashUsage

func NormalizeMode(mode Mode) Mode {
	return modes.Normalize(mode)
}

func SlashNext(input string, current Mode) (Mode, bool) {
	return modes.SlashNext(input, current)
}

func ToggledMode(current Mode) Mode {
	return modes.Toggled(current)
}
