package spinner

import (
	"strings"
	"testing"
)

func TestFramesAllKinds(t *testing.T) {
	kinds := []Kind{Dots, Bounce, Grow, Arrows, Star, Moon, Pulse, Brain, Sparkle}
	for _, k := range kinds {
		f := Frames(k)
		if len(f) == 0 {
			t.Fatalf("Frames(%s) empty", k)
		}
	}
}

func TestFramesUnknownDefaults(t *testing.T) {
	f := Frames("nonexistent")
	if len(f) != len(Frames(Dots)) {
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

func TestWingsForSkinAres(t *testing.T) {
	w := WingsForSkin("ares")
	if len(w) != 4 {
		t.Fatalf("ares wings count = %d, want 4", len(w))
	}
	if w[0].Left != "⟪⚔" || w[0].Right != "⚔⟫" {
		t.Fatalf("ares wings[0] = %+v", w[0])
	}
}

func TestWingsForSkinDefault(t *testing.T) {
	w := WingsForSkin("default")
	if len(w) != 0 {
		t.Fatal("default skin should have no wings")
	}
}

func TestRender(t *testing.T) {
	out := Render(Dots, 0, 0, 0, 0, "ares", "1.2")
	if !strings.Contains(out, "⟪⚔") || !strings.Contains(out, "1.2s") {
		t.Fatalf("Render = %q", out)
	}
}
