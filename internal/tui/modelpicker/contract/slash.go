package contract

import "strings"

// SlashArgument returns the free-form model argument after the slash command.
func SlashArgument(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	fields := strings.Fields(trimmed)
	if len(fields) <= 1 {
		return ""
	}
	idx := strings.Index(trimmed, fields[1])
	if idx < 0 {
		return strings.Join(fields[1:], " ")
	}
	return strings.TrimSpace(trimmed[idx:])
}
