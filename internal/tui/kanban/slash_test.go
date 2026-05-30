package kanban

import (
	"strings"
	"testing"
)

func TestBoundStatusTrimsAndBounds(t *testing.T) {
	if got := BoundStatus(" task\n done "); got != "task done" {
		t.Fatalf("BoundStatus whitespace = %q, want collapsed", got)
	}
	long := strings.Repeat("x", MaxStatusRunes+10)
	got := BoundStatus(long)
	if len([]rune(got)) != MaxStatusRunes+len("...") {
		t.Fatalf("bounded rune count = %d, want %d", len([]rune(got)), MaxStatusRunes+len("..."))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("bounded status missing suffix: %q", got)
	}
}
