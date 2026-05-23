package tui

import (
	"fmt"
	"strings"
)

const indicatorUsage = "usage: /indicator [ascii|emoji|kaomoji|unicode]"

func indicatorSlashHandler(input string, model *Model) SlashResult {
	args := strings.Fields(strings.TrimSpace(input))
	if len(args) <= 1 {
		return SlashResult{Handled: true, StatusMessage: fmt.Sprintf("indicator: %s", NormalizeIndicatorStyle(string(model.indicatorStyle)))}
	}
	if len(args) > 2 {
		return SlashResult{Handled: true, StatusMessage: indicatorUsage}
	}
	style := strings.ToLower(strings.TrimSpace(args[1]))
	if NormalizeIndicatorStyle(style) != IndicatorStyle(style) {
		return SlashResult{Handled: true, StatusMessage: indicatorUsage}
	}
	model.indicatorStyle = IndicatorStyle(style)
	return SlashResult{Handled: true, StatusMessage: fmt.Sprintf("indicator → %s", model.indicatorStyle)}
}
