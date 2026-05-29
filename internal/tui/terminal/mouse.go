package terminal

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const MouseSlashUsage = "usage: /mouse [on|off|toggle]"

type MouseSlashResult struct {
	Handled bool
	Valid   bool
	Next    bool
	Message string
}

func ParseMouseTrackingSlash(input string, current bool) MouseSlashResult {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return MouseSlashResult{}
	}

	name := strings.ToLower(fields[0])
	if name != "/mouse" && name != "/scroll" {
		return MouseSlashResult{}
	}
	if len(fields) > 2 {
		return MouseSlashResult{Handled: true, Message: MouseSlashUsage}
	}

	arg := ""
	if len(fields) == 2 {
		arg = strings.ToLower(fields[1])
	}

	switch arg {
	case "", "toggle":
		return MouseSlashResult{Handled: true, Valid: true, Next: !current}
	case "on":
		return MouseSlashResult{Handled: true, Valid: true, Next: true}
	case "off":
		return MouseSlashResult{Handled: true, Valid: true, Next: false}
	default:
		return MouseSlashResult{Handled: true, Message: MouseSlashUsage}
	}
}

func DefaultMouseModeCmd(enabled bool) tea.Cmd {
	if enabled {
		return tea.EnableMouseAllMotion
	}
	return tea.DisableMouse
}
