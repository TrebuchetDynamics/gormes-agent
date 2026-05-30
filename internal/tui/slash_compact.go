package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/compact"

func compactSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "compact: TUI unavailable"}
	}
	next, ok := compactSlashNext(input, model.compactTranscript)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: compact.Usage}
	}
	model.compactTranscript = next
	if next {
		return SlashResult{Handled: true, StatusMessage: "compact on"}
	}
	return SlashResult{Handled: true, StatusMessage: "compact off"}
}

func compactSlashNext(input string, current bool) (bool, bool) {
	return compact.Next(input, current)
}
