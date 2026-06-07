package composer

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer/fastecho"

func CanFastAppendShape(current string, cursor int, text string, columns int, currentLineWidth int) bool {
	return fastecho.CanFastAppendShape(current, cursor, text, columns, currentLineWidth)
}

func CanFastBackspaceShape(current string, cursor int, columns ...int) bool {
	return fastecho.CanFastBackspaceShape(current, cursor, columns...)
}

func SupportsFastEchoTerminal(env map[string]string) bool {
	return fastecho.SupportsFastEchoTerminal(env)
}
