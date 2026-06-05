package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"

const OSC52ClipboardQuery = terminal.OSC52ClipboardQuery

type OSC52Response = terminal.OSC52Response
type OSC52SendFunc = terminal.OSC52SendFunc

func BuildOSC52ClipboardQuery(env map[string]string) string {
	return terminal.BuildOSC52ClipboardQuery(env)
}

func ParseOSC52ClipboardData(data string) (string, bool) {
	return terminal.ParseOSC52ClipboardData(data)
}

func ReadOSC52Clipboard(env map[string]string, send OSC52SendFunc, flush func() error) ClipboardTextResult {
	return terminal.ReadOSC52Clipboard(env, send, flush)
}
