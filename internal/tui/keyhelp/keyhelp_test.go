package keyhelp

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func TestRenderBarCleansEmptyItems(t *testing.T) {
	out := RenderBar(80, Styles{}, []Item{
		{Keys: []string{" ", "enter"}, Description: "choose"},
		{Keys: []string{"esc"}, Description: " "},
	})
	if !strings.Contains(out, "enter") || !strings.Contains(out, "choose") {
		t.Fatalf("RenderBar() = %q, want cleaned enter binding", out)
	}
	if strings.Contains(out, "esc") {
		t.Fatalf("RenderBar() = %q, should skip empty descriptions", out)
	}
}

func TestRenderBarWidthGate(t *testing.T) {
	if got := RenderBar(10, Styles{}, []Item{{Keys: []string{"x"}, Description: "do"}}); got != "" {
		t.Fatalf("RenderBar narrow = %q, want empty", got)
	}
}

func TestRenderBarUsesWidth(t *testing.T) {
	got := RenderBar(44, Styles{}, []Item{
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

func TestRenderBindingBarUsesBubblesKeyBindings(t *testing.T) {
	got := RenderBindingBar(60, Styles{}, []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "send")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "complete")),
	})
	for _, want := range []string{"Enter", "send", "Tab", "complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("binding help bar missing %q:\n%s", want, got)
		}
	}
}

func TestRenderBarDropsEmptyAndNarrow(t *testing.T) {
	if got := RenderBar(10, Styles{}, []Item{{Keys: []string{"Enter"}, Description: "send"}}); got != "" {
		t.Fatalf("narrow help bar = %q, want empty", got)
	}
	if got := RenderBar(80, Styles{}, []Item{{Keys: []string{""}, Description: "send"}}); got != "" {
		t.Fatalf("empty key help bar = %q, want empty", got)
	}
}
