package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSpinnerRenderCompatibilityWrapper(t *testing.T) {
	out := SpinnerRender(SpinnerDots, 0, 0, 0, 0, "ares", "1.2")
	if !strings.Contains(out, "⟪⚔") || !strings.Contains(out, "1.2s") {
		t.Fatalf("SpinnerRender = %q", out)
	}
}

func TestRenderDiff(t *testing.T) {
	skin := DefaultHermesSkin()
	out := RenderDiff(skin, "--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new", 0)
	if !strings.Contains(out, "old") || !strings.Contains(out, "new") {
		t.Fatalf("RenderDiff = %q", out)
	}
}

func TestRenderDiffUsesSharedSkinStyles(t *testing.T) {
	forceLipglossTrueColor(t)
	skin := BuiltinSkins()["poseidon"]
	shared := SkinStylesFor(skin)
	if got, want := shared.Good.GetForeground(), lipgloss.Color(skin.Colors.StatusBarGood); got != want {
		t.Fatalf("shared good foreground = %v, want %v", got, want)
	}
	if got, want := shared.Bad.GetForeground(), lipgloss.Color(skin.Colors.StatusBarBad); got != want {
		t.Fatalf("shared bad foreground = %v, want %v", got, want)
	}

	out := RenderDiff(skin, "@@ -1 +1 @@\n-old\n+new", 0)
	if !strings.Contains(out, "old") || !strings.Contains(out, "new") || !strings.Contains(out, "\x1b[") {
		t.Fatalf("styled diff should include visible diff text and skin ANSI styling; got %q", out)
	}
}
