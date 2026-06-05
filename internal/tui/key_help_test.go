package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

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
