package queue

import (
	"strings"
	"testing"
)

func TestRenderWidgetShowsEditedWindowAndDefaultsLabel(t *testing.T) {
	q := Messages{}
	q.Enqueue("alpha")
	q.Enqueue(" beta ")
	q.Enqueue("gamma")
	q.Enqueue("delta")
	q.Enqueue("epsilon")
	if !q.SelectEdit(2) {
		t.Fatalf("SelectEdit(2) = false, want true")
	}

	got := RenderWidget("", q, 80, nil)
	want := strings.Join([]string{
		"queued (5)",
		"  …",
		"  2. beta",
		"▸ 3. gamma",
		"  4. delta",
		"  … and 1 more",
	}, "\n")
	if got != want {
		t.Fatalf("RenderWidget() =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderWidgetAppliesTrimFuncPerLine(t *testing.T) {
	q := Messages{}
	q.Enqueue("long draft")

	got := RenderWidget("queued", q, 4, func(line string, width int) string {
		if len(line) <= width {
			return line
		}
		return line[:width]
	})
	want := "queu\n  1."
	if got != want {
		t.Fatalf("RenderWidget() = %q, want %q", got, want)
	}
}

func TestRenderWidgetEmptyCases(t *testing.T) {
	q := Messages{}
	q.Enqueue("hidden")
	if got := RenderWidget("queued", q, 0, nil); got != "" {
		t.Fatalf("RenderWidget(width=0) = %q, want empty", got)
	}
	if got := RenderWidget("queued", Messages{}, 80, nil); got != "" {
		t.Fatalf("RenderWidget(empty) = %q, want empty", got)
	}
}
