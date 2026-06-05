package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator"

const indicatorUsage = indicator.SlashUsage

func indicatorSlashHandler(input string, model *Model) SlashResult {
	result := indicator.ParseSlash(input, indicator.Style(model.indicatorStyle))
	if result.Apply {
		model.indicatorStyle = IndicatorStyle(result.Style)
	}
	return SlashResult{Handled: true, StatusMessage: result.Status}
}
