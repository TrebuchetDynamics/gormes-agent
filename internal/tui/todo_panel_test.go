package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderTodoPanelCompatibilityWrapper(t *testing.T) {
	items := []TodoItem{{Text: "Write tests", Status: TodoStatusPending, Collapsed: true}}
	got := RenderTodoPanel(items, 80)
	for _, want := range []string{"Write tests", "○", "▸"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderTodoPanel wrapper missing %q: %q", want, got)
		}
	}
	if TodoStatusDone.Glyph() != "●" {
		t.Fatalf("TodoStatusDone.Glyph() = %q, want ●", TodoStatusDone.Glyph())
	}
}

func TestRenderTodoPanelWithSkinUsesSharedStyles(t *testing.T) {
	forceLipglossTrueColor(t)
	skin := BuiltinSkins()["poseidon"]
	items := []TodoItem{
		{Text: "Design shared TUI styles", Status: TodoStatusPending},
		{Text: "Retheme done rows", Status: TodoStatusDone, Collapsed: true},
	}

	got := RenderTodoPanelWithSkin(items, 48, skin)
	for _, want := range []string{"○", "●", "Design shared TUI styles", "Retheme done rows", "▸"} {
		if !strings.Contains(got, want) {
			t.Fatalf("styled todo panel missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("styled todo panel should use active skin styles; got no ANSI styling:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 48 {
			t.Fatalf("styled todo panel line width %d exceeds 48: %q\n\n%s", w, line, got)
		}
	}
}

func TestModel_RenderTodoPanel(t *testing.T) {
	m := Model{
		todoReader: func(sessionID string) []TodoItem {
			if sessionID == "s1" {
				return []TodoItem{
					{Text: "Task one", Status: TodoStatusPending},
					{Text: "Task two", Status: TodoStatusDone},
				}
			}
			return nil
		},
		sessionID: "s1",
	}

	got := m.renderTodoPanel(80)
	if !strings.Contains(got, "Task one") {
		t.Errorf("renderTodoPanel should contain 'Task one', got:\n%s", got)
	}
	if !strings.Contains(got, "Task two") {
		t.Errorf("renderTodoPanel should contain 'Task two', got:\n%s", got)
	}

	m.sessionID = "s2"
	got = m.renderTodoPanel(80)
	if got != "" {
		t.Errorf("renderTodoPanel for empty session should be empty, got:\n%s", got)
	}

	m.todoReader = nil
	m.sessionID = "s1"
	got = m.renderTodoPanel(80)
	if got != "" {
		t.Errorf("renderTodoPanel with nil reader should be empty, got:\n%s", got)
	}
}
