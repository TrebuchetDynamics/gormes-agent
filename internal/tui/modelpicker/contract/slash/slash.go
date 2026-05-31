package slash

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/textinput"
)

// Argument returns the free-form model argument after the slash command.
func Argument(input string) string {
	trimmed := textinput.TrimBoundary(input)
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
	return textinput.TrimBoundary(trimmed[idx:])
}
