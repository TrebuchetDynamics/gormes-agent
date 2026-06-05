package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer"

// CanFastAppendShape mirrors Hermes' conservative fast-echo append gate. It
// only allows one-line printable ASCII writes that cannot wrap the prompt.
func CanFastAppendShape(current string, cursor int, text string, columns int, currentLineWidth int) bool {
	return composer.CanFastAppendShape(current, cursor, text, columns, currentLineWidth)
}

// CanFastBackspaceShape mirrors Hermes' conservative fast-echo delete gate.
// The optional columns argument preserves legacy behavior when omitted.
func CanFastBackspaceShape(current string, cursor int, columns ...int) bool {
	return composer.CanFastBackspaceShape(current, cursor, columns...)
}

func SupportsFastEchoTerminal(env map[string]string) bool {
	return composer.SupportsFastEchoTerminal(env)
}
