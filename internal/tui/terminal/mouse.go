package terminal

import (
	tea "github.com/charmbracelet/bubbletea"

	mousepkg "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/mouse"
)

const MouseSlashUsage = mousepkg.MouseSlashUsage

type MouseSlashResult = mousepkg.MouseSlashResult
type MouseSlashDecision = mousepkg.MouseSlashDecision

func HandleMouseSlash(input string, current bool) MouseSlashDecision {
	return mousepkg.HandleMouseSlash(input, current)
}

func ParseMouseTrackingSlash(input string, current bool) MouseSlashResult {
	return mousepkg.ParseMouseTrackingSlash(input, current)
}

func DefaultMouseModeCmd(enabled bool) tea.Cmd {
	return mousepkg.DefaultMouseModeCmd(enabled)
}
