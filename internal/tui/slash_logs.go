package tui

import (
	"strconv"
	"strings"
)

func logsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "logs: TUI unavailable"}
	}
	if model.gatewayLogTail == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no gateway logs"}
	}
	text, err := model.gatewayLogTail(logsTailLimit(input))
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "logs: " + err.Error()}
	}
	text = strings.TrimRight(text, "\n")
	if strings.TrimSpace(text) == "" {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no gateway logs"}
	}
	page := TransientPageState{Title: "Logs", Body: text}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "logs opened"}
}

func logsTailLimit(input string) int {
	fields := strings.Fields(strings.TrimSpace(input))
	limit := 20
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[1]); err == nil {
			switch {
			case n < 0:
				limit = 1
			case n > 0:
				limit = n
			}
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 80 {
		limit = 80
	}
	return limit
}
