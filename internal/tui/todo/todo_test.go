package todo

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderEmpty(t *testing.T) {
	if got := Render(nil, 80); got != "" {
		t.Fatalf("Render(nil) = %q, want empty", got)
	}
	if got := Render([]Item{}, 80); got != "" {
		t.Fatalf("Render(empty) = %q, want empty", got)
	}
}

func TestRenderGlyphsAndCollapsedMarker(t *testing.T) {
	got := Render([]Item{
		{Text: "Write tests", Status: StatusPending},
		{Text: "Ship", Status: StatusDone, Collapsed: true},
	}, 80)
	for _, want := range []string{"○", "Write tests", "●", "Ship", "▸"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Render() missing %q:\n%s", want, got)
		}
	}
}

func TestRenderWidthRespected(t *testing.T) {
	got := Render([]Item{{Text: "A very long task that should be truncated", Status: StatusPending}}, 20)
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 20 {
			t.Fatalf("line width %d exceeds 20: %q", width, line)
		}
	}
}

func TestRenderWithStyles(t *testing.T) {
	got := RenderWithStyles([]Item{{Text: "Done", Status: StatusDone, Collapsed: true}}, 80, Styles{
		Accent: wrap("accent"),
		Good:   wrap("good"),
		Dim:    wrap("dim"),
		Text:   wrap("text"),
	})
	for _, want := range []string{"[good]●[/good]", "[dim]▸[/dim]", "[dim]Done[/dim]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderWithStyles() missing %q:\n%s", want, got)
		}
	}
}

func wrap(label string) func(string) string {
	return func(s string) string { return "[" + label + "]" + s + "[/" + label + "]" }
}
