package tui

import "strings"

const detailsSlashUsage = "usage: /details [hidden|collapsed|expanded|cycle]  or  /details <section> [hidden|collapsed|expanded|reset]"
const detailsSectionSlashUsage = "usage: /details <section> [hidden|collapsed|expanded|reset]"

func detailsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "details: TUI unavailable"}
	}
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		model.detailsState = NormalizeDetailsState(model.detailsState)
		return SlashResult{Handled: true, StatusMessage: model.detailsState.Status()}
	}
	first := strings.ToLower(fields[1])
	if section, ok := ParseDetailsSection(first); ok {
		if len(fields) != 3 {
			return SlashResult{Handled: true, StatusMessage: detailsSectionSlashUsage}
		}
		second := strings.ToLower(fields[2])
		if second == "reset" || second == "clear" || second == "default" {
			model.detailsState = model.detailsState.WithoutSection(section)
			return SlashResult{Handled: true, StatusMessage: "details " + string(section) + ": reset"}
		}
		mode, ok := ParseDetailsMode(second)
		if !ok {
			return SlashResult{Handled: true, StatusMessage: detailsSectionSlashUsage}
		}
		model.detailsState = model.detailsState.WithSection(section, mode)
		return SlashResult{Handled: true, StatusMessage: "details " + string(section) + ": " + string(mode)}
	}
	if len(fields) != 2 {
		return SlashResult{Handled: true, StatusMessage: detailsSlashUsage}
	}
	var next DetailsMode
	switch first {
	case "cycle", "toggle":
		next = NextDetailsMode(NormalizeDetailsState(model.detailsState).Global)
	default:
		mode, ok := ParseDetailsMode(first)
		if !ok {
			return SlashResult{Handled: true, StatusMessage: detailsSlashUsage}
		}
		next = mode
	}
	model.detailsState = model.detailsState.WithGlobal(next)
	return SlashResult{Handled: true, StatusMessage: "details: " + string(next)}
}
