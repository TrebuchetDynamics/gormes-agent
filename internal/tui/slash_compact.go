package tui

import "strings"

func compactSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "compact: TUI unavailable"}
	}
	next, ok := compactSlashNext(input, model.compactTranscript)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: "usage: /compact [on|off|toggle]"}
	}
	model.compactTranscript = next
	if next {
		return SlashResult{Handled: true, StatusMessage: "compact on"}
	}
	return SlashResult{Handled: true, StatusMessage: "compact off"}
}

func compactSlashNext(input string, current bool) (bool, bool) {
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
