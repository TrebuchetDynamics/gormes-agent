package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func TestRenderKeyHelpBarUsesSharedStylesAndWidth(t *testing.T) {
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

func TestRenderKeyBindingHelpBarUsesBubblesKeyBindings(t *testing.T) {
	got := RenderKeyBindingHelpBar(60, DefaultHermesSkin(), []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "send")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "complete")),
	})
	for _, want := range []string{"Enter", "send", "Tab", "complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("binding help bar missing %q:\n%s", want, got)
		}
	}
}

func TestRenderKeyHelpBarDropsEmptyAndNarrow(t *testing.T) {
	if got := RenderKeyHelpBar(10, DefaultHermesSkin(), []KeyHelp{{Keys: []string{"Enter"}, Description: "send"}}); got != "" {
		t.Fatalf("narrow help bar = %q, want empty", got)
	}
	if got := RenderKeyHelpBar(80, DefaultHermesSkin(), []KeyHelp{{Keys: []string{""}, Description: "send"}}); got != "" {
		t.Fatalf("empty key help bar = %q, want empty", got)
	}
}
