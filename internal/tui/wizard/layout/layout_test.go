package layout

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestWrapIndentedKeepsContinuationAligned(t *testing.T) {
	got := WrapIndented("→ ", "alpha beta gamma", 10)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("WrapIndented did not wrap: %q", got)
	}
	if !strings.HasPrefix(lines[0], "→ ") {
		t.Fatalf("first line missing prefix: %q", got)
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("continuation line not aligned: %q", got)
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 10 {
			t.Fatalf("line width %d exceeds 10: %q", width, line)
		}
	}
}

func TestClampViewKeepsPromptFocalAndHelp(t *testing.T) {
	got := ClampView("title\n\nPrompt text\n\n> value\n\nEnter submit  Esc abort", 20, 5)
	for _, want := range []string{"title", "omitted", "> value", "Enter submit"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ClampView missing %q:\n%s", want, got)
		}
	}
	if lines := strings.Split(got, "\n"); len(lines) > 5 {
		t.Fatalf("ClampView height = %d, want <= 5:\n%s", len(lines), got)
	}
}

func TestTrimToWidthAddsEllipsisWithinDisplayWidth(t *testing.T) {
	got := TrimToWidth("abcdef", 4)
	if got != "abc…" {
		t.Fatalf("TrimToWidth = %q, want abc…", got)
	}
	if width := lipgloss.Width(got); width > 4 {
		t.Fatalf("TrimToWidth width = %d, want <= 4", width)
	}
}
