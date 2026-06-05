package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/kanban"

// KanbanSlashFunc runs a full /kanban editor command and returns bounded
// operator-facing output. Production binds this to cmd/gormes' Kanban Cobra
// tree; tests can inject a fake without opening a database.
type KanbanSlashFunc func(input string) (string, error)

const maxKanbanSlashStatusRunes = kanban.MaxStatusRunes

func kanbanSlashHandler(input string, model *Model) SlashResult {
	var run kanban.Runner
	if model != nil && model.kanbanSlash != nil {
		run = kanban.Runner(model.kanbanSlash)
	}
	result := kanban.HandleSlash(input, run)
	return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
}

func boundKanbanSlashStatus(status string) string {
	return kanban.BoundStatus(status)
}
