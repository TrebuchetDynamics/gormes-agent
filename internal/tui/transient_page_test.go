package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func forceLipglossTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func TestRenderTransientPageWithSkinUsesSharedChrome(t *testing.T) {
	forceLipglossTrueColor(t)
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
