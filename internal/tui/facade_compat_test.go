package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/testenv"
)

func TestTUIANSICompatibilityWrappers(t *testing.T) {
	input := "A\x1b[31mB\x1b[2JC"
	if got := StripANSIForTUI(input); got != "ABC" {
		t.Fatalf("StripANSIForTUI() = %q, want %q", got, "ABC")
	}
	if got := SanitizeANSIForRender(input); got != "A\x1b[31mBC" {
		t.Fatalf("SanitizeANSIForRender() = %q", got)
	}
	if !HasANSI(input) {
		t.Fatal("HasANSI() = false, want true")
	}
}

func TestBuildAutoTitleRequestCompatibilityWrapper(t *testing.T) {
	in := AutoTitleInput{
		SessionKey:    "session-key-1",
		Status:        "complete",
		UserText:      "  hello there  ",
		AssistantText: "  general kenobi  ",
		HistoryCount:  2,
	}

	got, ok := BuildAutoTitleRequest(in)
	if !ok {
		t.Fatalf("BuildAutoTitleRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "session-key-1" || got.UserText != in.UserText || got.AssistantText != in.AssistantText || got.HistoryCount != 2 {
		t.Fatalf("BuildAutoTitleRequest(%+v) = (%+v, true); wrapper did not preserve autotitle request", in, got)
	}
}

func TestTUICompletionRequestCompatibilityWrapper(t *testing.T) {
	got, ok := CompletionRequestForInput("/help")
	if !ok {
		t.Fatal("CompletionRequestForInput(/help) ok = false, want true")
	}
	want := TUICompletionRequest{Method: TUICompletionSlash, Text: "/help", ReplaceFrom: 1}
	if got != want {
		t.Fatalf("CompletionRequestForInput(/help) = %+v, want %+v", got, want)
	}
}

func TestRenderKeyHelpBarCompatibilityWrapperUsesSharedStylesAndWidth(t *testing.T) {
	got := RenderKeyHelpBar(44, DefaultHermesSkin(), []KeyHelp{
		{Keys: []string{"Enter"}, Description: "send"},
		{Keys: []string{"Shift+Enter", "Ctrl+J"}, Description: "newline"},
	})

	for _, want := range []string{"Enter", "send", "Shift+Enter/Ctrl+J", "newline"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help bar missing %q:\n%s", want, got)
		}
	}
	if width := lipgloss.Width(got); width > 44 {
		t.Fatalf("help bar width %d exceeds 44: %q", width, got)
	}
}

func TestRunningPlaceholderCompatibilityWrapper(t *testing.T) {
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.inFlight = false
	if got := m.RunningPlaceholder(); got != "Type a message and hit Enter…" {
		t.Fatalf("idle RunningPlaceholder() = %q", got)
	}

	registry := NewSlashRegistry()
	noop := func(string, *Model) SlashResult { return SlashResult{Handled: true} }
	registry.Register("queue", noop, WithBusyAvailable())
	registry.Register("steer", noop, WithBusyAvailable())
	m.slashRegistry = registry
	m.inFlight = true
	if got := m.RunningPlaceholder(); got != "msg=interrupt · /queue · /steer · Ctrl+C cancel" {
		t.Fatalf("busy RunningPlaceholder() = %q", got)
	}
}

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
	testenv.TrueColor(t)
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

func TestRenderTransientPageWithSkinUsesSharedChrome(t *testing.T) {
	testenv.TrueColor(t)
	skin := BuiltinSkins()["poseidon"]
	page := TransientPageState{Title: "Status", Body: "active profile\nready"}

	got := RenderTransientPageWithSkin(page, 44, 8, skin)
	for _, want := range []string{"Status", "active profile", "Esc to close"} {
		if !strings.Contains(got, want) {
			t.Fatalf("transient page missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("transient page should use active skin styles; got no ANSI styling:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 44 {
			t.Fatalf("transient page line width %d exceeds 44: %q\n\n%s", w, line, got)
		}
	}
}
