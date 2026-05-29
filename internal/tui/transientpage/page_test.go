package transientpage

import (
	"strings"
	"testing"
)

func TestRenderDefaultsAndEmptyBody(t *testing.T) {
	got := Render(State{}, 10, 0, Styles{}, false, nil)
	for _, want := range []string{"╭─ Page", "│ (empty)", "╰─ Esc to close"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Render missing %q:\n%s", want, got)
		}
	}
}

func TestRenderAppliesOffsetAndHeight(t *testing.T) {
	page := State{Title: "Logs", Body: "one\ntwo\nthree\nfour", Offset: 1}
	got := Render(page, 80, 2, Styles{}, false, nil)
	if strings.Contains(got, "one") || strings.Contains(got, "four") {
		t.Fatalf("Render did not apply offset/height:\n%s", got)
	}
	if !strings.Contains(got, "two") || !strings.Contains(got, "more lines") {
		t.Fatalf("Render missing expected visible lines:\n%s", got)
	}
}

func TestLinesUsesWrapper(t *testing.T) {
	wrap := func(line string, width int) string {
		if width != 20 && width != 12 {
			t.Fatalf("unexpected width %d", width)
		}
		return strings.ReplaceAll(line, " ", "\n")
	}
	got := Lines("alpha beta", 12, wrap)
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("Lines = %#v", got)
	}
}
