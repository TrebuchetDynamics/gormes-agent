package composer

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/envmap"
)

// CanFastAppendShape mirrors Hermes' conservative fast-echo append gate. It
// only allows one-line printable ASCII writes that cannot wrap the prompt.
func CanFastAppendShape(current string, cursor int, text string, columns int, currentLineWidth int) bool {
	if current == "" || cursor != len(current) || strings.Contains(current, "\n") {
		return false
	}
	if !isPrintableASCII(text) {
		return false
	}
	width := columns
	if width < 1 {
		width = 1
	}
	return currentLineWidth+len(text) < width
}

// CanFastBackspaceShape mirrors Hermes' conservative fast-echo delete gate.
// The optional columns argument preserves legacy behavior when omitted.
func CanFastBackspaceShape(current string, cursor int, columns ...int) bool {
	if current == "" || cursor <= 0 || cursor != len(current) || strings.Contains(current, "\n") {
		return false
	}
	if !isPrintableASCII(current) {
		return false
	}
	if len(columns) > 0 {
		width := columns[0]
		if width < 1 {
			width = 1
		}
		if cursor%width == 0 {
			return false
		}
	}
	return true
}

func SupportsFastEchoTerminal(env map[string]string) bool {
	return envmap.Value(env, "TERM_PROGRAM") != "Apple_Terminal"
}

func isPrintableASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
