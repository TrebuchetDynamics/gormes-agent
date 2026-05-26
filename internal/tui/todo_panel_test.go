package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTodoItem_Struct(t *testing.T) {
	item := TodoItem{
		Text:      "Test task",
		Status:    TodoStatusPending,
		Collapsed: false,
	}
	if item.Text != "Test task" {
		t.Fatalf("expected Text 'Test task', got %q", item.Text)
	}
	if item.Status != TodoStatusPending {
		t.Fatalf("expected Status TodoStatusPending, got %v", item.Status)
	}
	if item.Collapsed {
		t.Fatal("expected Collapsed false")
	}
}

func TestTodoStatus_Glyphs(t *testing.T) {
	if TodoStatusPending.Glyph() != "○" {
		t.Fatalf("pending glyph = %q, want ○", TodoStatusPending.Glyph())
	}
	if TodoStatusDone.Glyph() != "●" {
		t.Fatalf("done glyph = %q, want ●", TodoStatusDone.Glyph())
	}
}

func TestRenderTodoPanel_Empty(t *testing.T) {
	got := RenderTodoPanel(nil, 80)
	if got != "" {
		t.Fatalf("empty panel = %q, want empty string", got)
	}

	got = RenderTodoPanel([]TodoItem{}, 80)
	if got != "" {
		t.Fatalf("empty slice = %q, want empty string", got)
	}
}

func TestRenderTodoPanel_SingleItem(t *testing.T) {
	items := []TodoItem{
		{Text: "Write tests", Status: TodoStatusPending, Collapsed: false},
	}
	got := RenderTodoPanel(items, 80)

	if !strings.Contains(got, "Write tests") {
		t.Fatalf("panel missing item text: %q", got)
	}
	if !strings.Contains(got, "○") {
		t.Fatalf("panel missing pending glyph: %q", got)
	}
}

func TestRenderTodoPanel_DoneItem(t *testing.T) {
	items := []TodoItem{
		{Text: "Done task", Status: TodoStatusDone, Collapsed: false},
	}
	got := RenderTodoPanel(items, 80)

	if !strings.Contains(got, "Done task") {
		t.Fatalf("panel missing done item text: %q", got)
	}
	if !strings.Contains(got, "●") {
		t.Fatalf("panel missing done glyph: %q", got)
	}
}

func TestRenderTodoPanel_CollapsedSection(t *testing.T) {
	items := []TodoItem{
		{Text: "Hidden task", Status: TodoStatusPending, Collapsed: true},
	}
	got := RenderTodoPanel(items, 80)

	// Collapsed items should show collapse indicator
	if !strings.Contains(got, "▸") && !strings.Contains(got, "▾") {
		t.Fatalf("collapsed panel missing collapse indicator: %q", got)
	}
}

func TestRenderTodoPanel_MultipleItems(t *testing.T) {
	items := []TodoItem{
		{Text: "First", Status: TodoStatusPending, Collapsed: false},
		{Text: "Second", Status: TodoStatusDone, Collapsed: false},
		{Text: "Third", Status: TodoStatusPending, Collapsed: false},
	}
	got := RenderTodoPanel(items, 80)

	for _, item := range items {
		if !strings.Contains(got, item.Text) {
			t.Fatalf("panel missing item text %q: %q", item.Text, got)
		}
	}
}

func TestRenderTodoPanel_WidthRespected(t *testing.T) {
	items := []TodoItem{
		{Text: "A very long task that should be truncated if needed", Status: TodoStatusPending, Collapsed: false},
	}

	// Should not panic at any width
	for _, width := range []int{0, 20, 40, 60, 80, 120} {
		got := RenderTodoPanel(items, width)
		if got == "" && width > 0 {
			t.Fatalf("width=%d: got empty string for non-empty input", width)
		}
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
