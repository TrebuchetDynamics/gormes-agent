package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSpinnerFrames_AllKinds(t *testing.T) {
	kinds := []SpinnerKind{SpinnerDots, SpinnerBounce, SpinnerGrow, SpinnerArrows, SpinnerStar, SpinnerMoon, SpinnerPulse, SpinnerBrain, SpinnerSparkle}
	for _, k := range kinds {
		f := SpinnerFrames(k)
		if len(f) == 0 {
			t.Fatalf("SpinnerFrames(%s) empty", k)
		}
	}
}

func TestSpinnerFrames_UnknownDefaults(t *testing.T) {
	f := SpinnerFrames("nonexistent")
	if len(f) != len(SpinnerFrames(SpinnerDots)) {
		t.Fatal("unknown spinner should default to dots")
	}
}

func TestWaitingFace(t *testing.T) {
	for i := range 50 {
		f := WaitingFace(i)
		if f == "" {
			t.Fatalf("WaitingFace(%d) empty", i)
		}
	}
}

func TestThinkingFace(t *testing.T) {
	for i := range 50 {
		f := ThinkingFace(i)
		if f == "" {
			t.Fatalf("ThinkingFace(%d) empty", i)
		}
	}
}

func TestThinkingVerb(t *testing.T) {
	for i := range 50 {
		v := ThinkingVerb(i)
		if v == "" {
			t.Fatalf("ThinkingVerb(%d) empty", i)
		}
	}
}

func TestSpinnerWings_Ares(t *testing.T) {
	w := SpinnerWingsForSkin("ares")
	if len(w) != 4 {
		t.Fatalf("ares wings count = %d, want 4", len(w))
	}
	if w[0].Left != "⟪⚔" || w[0].Right != "⚔⟫" {
		t.Fatalf("ares wings[0] = %+v", w[0])
	}
}

func TestSpinnerWings_Default(t *testing.T) {
	w := SpinnerWingsForSkin("default")
	if len(w) != 0 {
		t.Fatal("default skin should have no wings")
	}
}

func TestSpinnerRender(t *testing.T) {
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
