package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"
)

const mouseSlashUsage = terminal.MouseSlashUsage

type mouseSlashResult struct {
	handled bool
	valid   bool
	next    bool
	message string
}

func parseMouseTrackingSlash(input string, current bool) mouseSlashResult {
	result := terminal.ParseMouseTrackingSlash(input, current)
	return mouseSlashResult{
		handled: result.Handled,
		valid:   result.Valid,
		next:    result.Next,
		message: result.Message,
	}
}

func defaultMouseModeCmd(enabled bool) tea.Cmd {
	return terminal.DefaultMouseModeCmd(enabled)
}

func (m Model) emitMouseModeCmd(enabled bool) tea.Cmd {
	if m.mouseModeCmd != nil {
		return m.mouseModeCmd(enabled)
	}
	return defaultMouseModeCmd(enabled)
}
