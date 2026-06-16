package widget

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/buffer"
)

func TestRenderShowsEditedWindowAndDefaultsLabel(t *testing.T) {
	q := buffer.Messages{}
	q.Enqueue("alpha")
	q.Enqueue(" beta ")
	q.Enqueue("gamma")
	q.Enqueue("delta")
	q.Enqueue("epsilon")
	if !q.SelectEdit(2) {
		t.Fatalf("SelectEdit(2) = false, want true")
	}

	got := Render("", q, 80, nil)
	want := strings.Join([]string{
		"queued (5)",
		"  …",
		"  2. beta",
		"▸ 3. gamma",
		"  4. delta",
		"  … and 1 more",
	}, "\n")
	if got != want {
		t.Fatalf("Render() =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderAppliesTrimFuncPerLine(t *testing.T) {
	q := buffer.Messages{}
	q.Enqueue("long draft")

	got := Render("queued", q, 4, func(line string, width int) string {
		if len(line) <= width {
			return line
		}
		return line[:width]
	})
	want := "queu\n  1."
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderEmptyCases(t *testing.T) {
	q := buffer.Messages{}
	q.Enqueue("hidden")
	if got := Render("queued", q, 0, nil); got != "" {
		t.Fatalf("Render(width=0) = %q, want empty", got)
	}
	if got := Render("queued", buffer.Messages{}, 80, nil); got != "" {
		t.Fatalf("Render(empty) = %q, want empty", got)
	}
}
