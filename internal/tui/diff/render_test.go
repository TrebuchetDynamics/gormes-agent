package diff

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderClassifiesUnifiedDiffLines(t *testing.T) {
	styles := Styles{
		Minus: lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Plus:  lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Hunk:  lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		File:  lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	}
	out := Render(styles, "--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new\n context", 0)
	for _, want := range []string{"--- a/file.txt", "+++ b/file.txt", "@@ -1 +1 @@", "old", "new", "context"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Render output missing %q: %q", want, out)
		}
	}
}

func TestRenderHonorsMaxLines(t *testing.T) {
	out := Render(Styles{}, "one\ntwo\nthree", 2)
	if got, want := strings.Count(out, "\n"), 2; got != want {
		t.Fatalf("rendered newline count = %d, want %d in %q", got, want, out)
	}
	if strings.Contains(out, "three") {
		t.Fatalf("Render did not truncate: %q", out)
	}
}
