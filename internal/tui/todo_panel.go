// Package tui — Hermes-compatible TodoPanel renderer.
//
// This module preserves the public tui todo-panel seam while delegating pure
// rendering mechanics to internal/tui/todo.
package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/todo"

// TodoStatus represents the completion state of a todo item.
type TodoStatus = todo.Status

const (
	// TodoStatusPending indicates an incomplete task.
	TodoStatusPending TodoStatus = todo.StatusPending
	// TodoStatusDone indicates a completed task.
	TodoStatusDone TodoStatus = todo.StatusDone
)

// TodoItem represents a single task in the todo panel.
type TodoItem = todo.Item

// RenderTodoPanel renders a collapsible todo list for the given items.
// It returns an empty string if items is nil or empty.
// Width constrains the panel; items may be truncated to fit.
func RenderTodoPanel(items []TodoItem, width int) string {
	return todo.Render(items, width)
}

func RenderTodoPanelWithSkin(items []TodoItem, width int, skin HermesSkin) string {
	styles := SkinStylesFor(skin)
	return todo.RenderWithStyles(items, width, todo.Styles{
		Accent: func(text string) string { return styles.Accent.Render(text) },
		Good:   func(text string) string { return styles.Good.Render(text) },
		Dim:    func(text string) string { return styles.Dim.Render(text) },
		Text:   func(text string) string { return styles.Text.Render(text) },
	})
}
