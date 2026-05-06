package cli

import (
	"strings"
	"testing"
)

func TestUserMessagePreviewDefaultHeadTail(t *testing.T) {
	got := FormatUserMessagePreview("line1\nline2\nline3\nline4\nline5\nline6", UserMessagePreviewConfig{})

	for _, want := range []string{"line1", "line2", "line5", "line6", "(+2 more lines)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"line3", "line4"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("preview leaked hidden line %q:\n%s", unwanted, got)
		}
	}
}

func TestUserMessagePreviewHideTail(t *testing.T) {
	got := FormatUserMessagePreview("line1\nline2\nline3\nline4\nline5\nline6", UserMessagePreviewConfig{
		FirstLines: 2,
		LastLines:  0,
	})

	for _, want := range []string{"line1", "line2", "(+4 more lines)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"line5", "line6"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("preview leaked tail line %q when tail disabled:\n%s", unwanted, got)
		}
	}
}

func TestUserMessagePreviewInvalidConfigFallsBack(t *testing.T) {
	t.Run("first lines lower bound mirrors Hermes", func(t *testing.T) {
		got := FormatUserMessagePreview("line1\nline2\nline3\nline4", UserMessagePreviewConfig{
			FirstLines: 0,
			LastLines:  2,
		})
		for _, want := range []string{"line1", "line3", "line4", "(+1 more line)"} {
			if !strings.Contains(got, want) {
				t.Fatalf("preview missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "line2") {
			t.Fatalf("preview leaked hidden middle line:\n%s", got)
		}
	})

	t.Run("non numeric values use defaults", func(t *testing.T) {
		got := FormatUserMessagePreview("line1\nline2\nline3\nline4\nline5", UserMessagePreviewConfig{
			FirstLines: "nope",
			LastLines:  "also-nope",
		})
		for _, want := range []string{"line1", "line2", "line4", "line5", "(+1 more line)"} {
			if !strings.Contains(got, want) {
				t.Fatalf("preview missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "line3") {
			t.Fatalf("preview leaked hidden middle line:\n%s", got)
		}
	})
}

func TestUserMessagePreviewShortInputNoMarker(t *testing.T) {
	got := FormatUserMessagePreview("line1\nline2\nline3", UserMessagePreviewConfig{
		FirstLines: 2,
		LastLines:  2,
	})

	for _, want := range []string{"line1", "line2", "line3"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "more line") {
		t.Fatalf("short preview should not include hidden marker:\n%s", got)
	}
}

func TestUserMessagePreviewEscapesMarkup(t *testing.T) {
	got := FormatUserMessagePreview("\x1b[31m[bold]secret[/]\x1b[0m\nnext", UserMessagePreviewConfig{})

	for _, unwanted := range []string{"\x1b", "[bold]", "[/]"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("preview leaked markup/control text %q:\n%q", unwanted, got)
		}
	}
	if !strings.Contains(got, `\[bold\]secret\[/\]`) {
		t.Fatalf("preview did not preserve escaped text:\n%q", got)
	}
}

func TestUserMessagePreviewSingleLineBullet(t *testing.T) {
	got := FormatUserMessagePreview("hello", UserMessagePreviewConfig{})

	if got != "● hello" {
		t.Fatalf("single-line preview = %q, want bullet line", got)
	}
	if strings.Contains(got, "more line") {
		t.Fatalf("single-line preview should not include hidden marker:\n%s", got)
	}
}
