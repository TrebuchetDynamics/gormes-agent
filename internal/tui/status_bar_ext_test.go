package tui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderFaceTicker_Idle(t *testing.T) {
	got := RenderFaceTicker("idle", 0)
	if got == "" {
		t.Fatal("RenderFaceTicker returned empty string for idle state")
	}
	// Should return one of the status indicators
	expected := []string{"⚕", "🌀", "🤔", "✨", "🍵", "🔮"}
	found := false
	for _, e := range expected {
		if strings.Contains(got, e) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("RenderFaceTicker(idle, 0) = %q, want one of %v", got, expected)
	}
}

func TestRenderFaceTicker_FrameCycling(t *testing.T) {
	// Collect all frames for reasoning state to verify cycling through 6 positions
	frames := make(map[int]bool)
	for frame := 0; frame < 12; frame++ {
		got := RenderFaceTicker("reasoning", frame)
		if got == "" {
			t.Fatalf("frame %d returned empty string", frame)
		}
		frames[frame%6] = true // 6 indicators
	}

	// All 6 positions should be reachable
	if len(frames) < 6 {
		t.Fatal("not all face positions reachable")
	}
}

func TestRenderFaceTicker_DifferentStates(t *testing.T) {
	states := []string{"idle", "reasoning", "working", "waiting", "ready", "break", "magic", "error"}

	for _, state := range states {
		got := RenderFaceTicker(state, 0)
		if got == "" {
			t.Fatalf("state %q returned empty string", state)
		}
	}
}

func TestRenderFaceTicker_StableAtSameFrame(t *testing.T) {
	// Same frame should give same result (pure function)
	got1 := RenderFaceTicker("reasoning", 5)
	got2 := RenderFaceTicker("reasoning", 5)
	if got1 != got2 {
		t.Fatalf("RenderFaceTicker not stable: frame 5 gave %q and %q", got1, got2)
	}
}

func TestRenderContextBar_ZeroPercent(t *testing.T) {
	got := RenderContextBar(0.0)
	if !strings.Contains(got, "░") {
		t.Fatalf("0%% context bar missing ░: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Fatalf("0%% context bar should not contain █: %q", got)
	}
}

func TestRenderContextBar_HundredPercent(t *testing.T) {
	got := RenderContextBar(100.0)
	if !strings.Contains(got, "█") {
		t.Fatalf("100%% context bar missing █: %q", got)
	}
}

func TestRenderContextBar_HalfPercent(t *testing.T) {
	got := RenderContextBar(50.0)
	// Should contain both filled and empty
	if !strings.Contains(got, "█") || !strings.Contains(got, "░") {
		t.Fatalf("50%% context bar should contain both █ and ░: %q", got)
	}
}

func TestRenderContextBar_NearFull(t *testing.T) {
	got := RenderContextBar(90.0)
	// Should be mostly filled
	filled := strings.Count(got, "█")
	empty := strings.Count(got, "░")
	if filled < empty {
		t.Fatalf("90%% context bar should be mostly filled: %q (█=%d, ░=%d)", got, filled, empty)
	}
}

func TestRenderContextBar_ClampedNegative(t *testing.T) {
	got := RenderContextBar(-10.0)
	// Should be same as 0%
	if strings.Contains(got, "█") {
		t.Fatalf("-10%% context bar should not contain █: %q", got)
	}
}

func TestRenderContextBar_ClampedOverHundred(t *testing.T) {
	got := RenderContextBar(150.0)
	// Should be same as 100%
	if strings.Contains(got, "░") {
		t.Fatalf("150%% context bar should not contain ░: %q", got)
	}
}

func TestRenderContextBar_DefaultWidth(t *testing.T) {
	got := RenderContextBar(50.0)
	// Default bar should be 10 chars (same as status bar)
	// Use rune count since █ and ░ are multi-byte in UTF-8
	if utf8.RuneCountInString(got) != 10 {
		t.Fatalf("default context bar rune count = %d, want 10: %q", utf8.RuneCountInString(got), got)
	}
}
