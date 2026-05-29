package statusbar

import "strings"

// Mode controls where the Hermes-compatible status rule renders.
type Mode string

const (
	ModeTop    Mode = "top"
	ModeBottom Mode = "bottom"
	ModeOff    Mode = "off"
)

const SlashUsage = "usage: /statusbar [on|off|top|bottom|toggle]"

func NormalizeMode(mode Mode) Mode {
	switch mode {
	case ModeTop, ModeBottom, ModeOff:
		return mode
	default:
		return ModeTop
	}
}

func SlashNext(input string, current Mode) (Mode, bool) {
	current = NormalizeMode(current)
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return ToggledMode(current), true
	}
	if len(fields) > 2 {
		return current, false
	}
	switch strings.ToLower(fields[1]) {
	case "on", "top":
		return ModeTop, true
	case "bottom":
		return ModeBottom, true
	case "off":
		return ModeOff, true
	case "toggle":
		return ToggledMode(current), true
	default:
		return current, false
	}
}

func ToggledMode(current Mode) Mode {
	if NormalizeMode(current) == ModeOff {
		return ModeTop
	}
	return ModeOff
}
