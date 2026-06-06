package statusbar

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderFaceTickerIdle(t *testing.T) {
	got := RenderFaceTicker("idle", 0)
	if got == "" {
		t.Fatal("RenderFaceTicker returned empty string for idle state")
	}
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

func TestRenderFaceTickerFrameCycling(t *testing.T) {
	frames := make(map[int]bool)
	for frame := 0; frame < 12; frame++ {
		got := RenderFaceTicker("reasoning", frame)
		if got == "" {
			t.Fatalf("frame %d returned empty string", frame)
		}
		frames[frame%6] = true
	}
	if len(frames) < 6 {
		t.Fatal("not all face positions reachable")
	}
}

func TestRenderFaceTickerDifferentStates(t *testing.T) {
	states := []string{"idle", "reasoning", "working", "waiting", "ready", "break", "magic", "error"}
	for _, state := range states {
		got := RenderFaceTicker(state, 0)
		if got == "" {
			t.Fatalf("state %q returned empty string", state)
		}
	}
}

func TestRenderFaceTickerStableAtSameFrame(t *testing.T) {
	got1 := RenderFaceTicker("reasoning", 5)
	got2 := RenderFaceTicker("reasoning", 5)
	if got1 != got2 {
		t.Fatalf("RenderFaceTicker not stable: frame 5 gave %q and %q", got1, got2)
	}
}

func TestRenderContextBar(t *testing.T) {
	tests := []struct {
		name       string
		pct        float64
		wantFilled int
		wantEmpty  int
	}{
		{name: "zero", pct: 0, wantFilled: 0, wantEmpty: 10},
		{name: "half", pct: 50, wantFilled: 5, wantEmpty: 5},
		{name: "near full", pct: 90, wantFilled: 9, wantEmpty: 1},
		{name: "negative clamps", pct: -10, wantFilled: 0, wantEmpty: 10},
		{name: "over hundred clamps", pct: 150, wantFilled: 10, wantEmpty: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderContextBar(tt.pct)
			if filled := strings.Count(got, "█"); filled != tt.wantFilled {
				t.Fatalf("filled count = %d, want %d in %q", filled, tt.wantFilled, got)
			}
			if empty := strings.Count(got, "░"); empty != tt.wantEmpty {
				t.Fatalf("empty count = %d, want %d in %q", empty, tt.wantEmpty, got)
			}
			if utf8.RuneCountInString(got) != ContextBarWidth {
				t.Fatalf("context bar rune count = %d, want %d: %q", utf8.RuneCountInString(got), ContextBarWidth, got)
			}
		})
	}
}

func TestRenderContextBarMatchesHermesFillPolicy(t *testing.T) {
	for _, pct := range []int{0, 5, 6, 49, 50, 94, 95, 100} {
		got := RenderContextBar(float64(pct))
		want := HermesContextBar(pct)
		if got != want {
			t.Fatalf("RenderContextBar(%d) = %q, want HermesContextBar(%d) %q", pct, got, pct, want)
		}
	}
}

func TestRenderContextBarWithLabelUsesClampedPercent(t *testing.T) {
	for _, tt := range []struct {
		name string
		pct  float64
		want string
	}{
		{name: "negative", pct: -10, want: "[░░░░░░░░░░] 0%"},
		{name: "over hundred", pct: 150, want: "[██████████] 100%"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderContextBarWithLabel(tt.pct); got != tt.want {
				t.Fatalf("RenderContextBarWithLabel(%v) = %q, want %q", tt.pct, got, tt.want)
			}
		})
	}
}
