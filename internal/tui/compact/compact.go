package compact

import "strings"

const Usage = "usage: /compact [on|off|toggle]"

type SlashResult struct {
	Next          bool
	OK            bool
	StatusMessage string
}

func HandleSlash(input string, current bool) SlashResult {
	next, ok := Next(input, current)
	if !ok {
		return SlashResult{Next: current, OK: false, StatusMessage: Usage}
	}
	if next {
		return SlashResult{Next: next, OK: true, StatusMessage: "compact on"}
	}
	return SlashResult{Next: next, OK: true, StatusMessage: "compact off"}
}

// Next resolves the compact-transcript state requested by a /compact command.
// It returns ok=false when the invocation should display Usage.
func Next(input string, current bool) (next bool, ok bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return !current, true
	}
	if len(fields) > 2 {
		return current, false
	}
	switch strings.ToLower(fields[1]) {
	case "on":
		return true, true
	case "off":
		return false, true
	case "toggle":
		return !current, true
	default:
		return current, false
	}
}
