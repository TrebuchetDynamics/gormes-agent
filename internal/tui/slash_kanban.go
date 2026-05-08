package tui

import (
	"strings"
	"unicode/utf8"
)

// KanbanSlashFunc runs a full /kanban editor command and returns bounded
// operator-facing output. Production binds this to cmd/gormes' Kanban Cobra
// tree; tests can inject a fake without opening a database.
type KanbanSlashFunc func(input string) (string, error)

const maxKanbanSlashStatusRunes = 600

func kanbanSlashHandler(input string, model *Model) SlashResult {
	if model == nil || model.kanbanSlash == nil {
		return SlashResult{Handled: true, StatusMessage: "kanban: command runner unavailable"}
	}
	output, err := model.kanbanSlash(input)
	status := strings.TrimSpace(output)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			msg = "command failed"
		}
		if status != "" {
			msg = msg + ": " + status
		}
		return SlashResult{Handled: true, StatusMessage: boundKanbanSlashStatus("kanban: " + msg)}
	}
	if status == "" {
		status = "kanban: no output"
	}
	return SlashResult{Handled: true, StatusMessage: boundKanbanSlashStatus(status)}
}

func boundKanbanSlashStatus(status string) string {
	status = strings.Join(strings.Fields(strings.TrimSpace(status)), " ")
	if utf8.RuneCountInString(status) <= maxKanbanSlashStatusRunes {
		return status
	}
	runes := []rune(status)
	return string(runes[:maxKanbanSlashStatusRunes]) + "..."
}
