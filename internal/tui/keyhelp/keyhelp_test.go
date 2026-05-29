package keyhelp

import (
	"strings"
	"testing"
)

func TestRenderBarCleansEmptyItems(t *testing.T) {
	out := RenderBar(80, Styles{}, []Item{
		{Keys: []string{" ", "enter"}, Description: "choose"},
		{Keys: []string{"esc"}, Description: " "},
	})
	if !strings.Contains(out, "enter") || !strings.Contains(out, "choose") {
		t.Fatalf("RenderBar() = %q, want cleaned enter binding", out)
	}
	if strings.Contains(out, "esc") {
		t.Fatalf("RenderBar() = %q, should skip empty descriptions", out)
	}
}

func TestRenderBarWidthGate(t *testing.T) {
	if got := RenderBar(10, Styles{}, []Item{{Keys: []string{"x"}, Description: "do"}}); got != "" {
		t.Fatalf("RenderBar narrow = %q, want empty", got)
	}
}
