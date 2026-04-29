package hermes

import (
	"strings"
	"testing"
)

func TestCompressionContentBudgetLength_ImageURLAddsFlatCharge(t *testing.T) {
	content := []any{
		map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": "data:image/png;base64,short"},
		},
	}
	if got := compressionContentBudgetLength(content); got != imageCharEquivalent {
		t.Fatalf("compressionContentBudgetLength(image_url) = %d, want %d", got, imageCharEquivalent)
	}
}

func TestCompressionContentBudgetLength_InputImageAndImageTypes(t *testing.T) {
	for _, kind := range []string{"input_image", "image"} {
		content := []any{map[string]any{"type": kind}}
		if got := compressionContentBudgetLength(content); got != imageCharEquivalent {
			t.Fatalf("compressionContentBudgetLength(%q) = %d, want %d", kind, got, imageCharEquivalent)
		}
	}
}

func TestCompressionContentBudgetLength_TextAndImagesSumsBoth(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "image_url"},
		map[string]any{"type": "input_image"},
		"world",
	}
	want := 5 + imageCharEquivalent + imageCharEquivalent + 5
	if got := compressionContentBudgetLength(content); got != want {
		t.Fatalf("compressionContentBudgetLength(text+images) = %d, want %d", got, want)
	}
}

func TestCompressionContentBudgetLength_RawBase64Ignored(t *testing.T) {
	huge := strings.Repeat("a", 1_000_000)
	content := []any{
		map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": "data:image/png;base64," + huge},
		},
	}
	if got := compressionContentBudgetLength(content); got != imageCharEquivalent {
		t.Fatalf("base64 dominated budget: got %d, want %d", got, imageCharEquivalent)
	}
}
