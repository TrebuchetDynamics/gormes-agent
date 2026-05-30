package tui

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statuspage"
)

func statusSlashHandler(_ string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "status: TUI unavailable"}
	}
	res := statuspage.HandleSlash(model.frame, model.SessionID())
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
	}
	model.transientPage = &res.Page
	return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
}

func BuildStatusPage(frame kernel.RenderFrame, sessionID string) TransientPageState {
	return statuspage.Build(frame, sessionID)
}
