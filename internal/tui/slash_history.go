package tui

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/historypage"
)

func historySlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "history: TUI unavailable"}
	}
	res := historypage.HandleSlash(model.frame.History, input)
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
	}
	model.transientPage = &res.Page
	return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
}

func historyPreviewLimit(input string) int {
	return historypage.PreviewLimit(input)
}

func BuildHistoryPage(history []llm.Message, preview int) (TransientPageState, bool) {
	return historypage.Build(history, preview)
}

func historyMessageText(msg llm.Message) string {
	return historypage.MessageText(msg)
}
