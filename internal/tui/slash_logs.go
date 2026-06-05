package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/logs"

func logsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "logs: TUI unavailable"}
	}
	res := logs.HandleSlash(input, func(limit int) (string, error) {
		if model.gatewayLogTail == nil {
			return "", nil
		}
		return model.gatewayLogTail(limit)
	})
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}
	page := TransientPageState{Title: res.Title, Body: res.Body}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func logsTailLimit(input string) int {
	return logs.TailLimit(input)
}
