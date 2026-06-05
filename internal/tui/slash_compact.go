package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/compact"

func compactSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "compact: TUI unavailable"}
	}
	result := compact.HandleSlash(input, model.compactTranscript)
	if result.OK {
		model.compactTranscript = result.Next
	}
	return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
}

func compactSlashNext(input string, current bool) (bool, bool) {
	return compact.Next(input, current)
}
