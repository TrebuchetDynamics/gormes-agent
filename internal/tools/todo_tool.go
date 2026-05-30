package tools

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/todo"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const TodoToolName = todo.ToolName

type TodoStatus = todo.Status

const (
	TodoStatusPending    TodoStatus = todo.StatusPending
	TodoStatusInProgress TodoStatus = todo.StatusInProgress
	TodoStatusCompleted  TodoStatus = todo.StatusCompleted
	TodoStatusCancelled  TodoStatus = todo.StatusCancelled
)

const (
	TodoEvidenceInvalidArgs      = todo.EvidenceInvalidArgs
	TodoEvidenceStoreUnavailable = todo.EvidenceStoreUnavailable
	TodoEvidenceStoreCorrupt     = todo.EvidenceStoreCorrupt
)

type TodoItem = todo.Item
type TodoSummary = todo.Summary
type TodoToolResult = todo.Result
type TodoStore = todo.Store
type TodoToolConfig = todo.Config
type TodoTool = todo.Tool

func NewTodoStore(root string) *TodoStore {
	return todo.NewStore(root)
}

func NewTodoTool(cfg TodoToolConfig) *TodoTool {
	return todo.NewTool(cfg)
}

func FormatTodoItemsForInjection(items []TodoItem) string {
	return todo.FormatItemsForInjection(items)
}

func NewTodoTools(cfg TodoToolConfig) []toolkit.Tool {
	return todo.NewTools(cfg)
}
