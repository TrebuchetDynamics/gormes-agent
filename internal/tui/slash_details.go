package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/details"

const detailsSlashUsage = details.SlashUsage
const detailsSectionSlashUsage = details.SectionSlashUsage

func detailsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "details: TUI unavailable"}
	}
	next, status := details.ApplySlash(input, model.detailsState)
	model.detailsState = next
	return SlashResult{Handled: true, StatusMessage: status}
}
