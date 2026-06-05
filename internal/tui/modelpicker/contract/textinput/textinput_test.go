package textinput

import "testing"

func TestTrimBoundaryPreservesCasingAndInternalSpacing(t *testing.T) {
	if got := TrimBoundary("  OPEN AI  "); got != "OPEN AI" {
		t.Fatalf("TrimBoundary = %q, want OPEN AI", got)
	}
}

func TestFirstNonEmptyReturnsFirstNormalizedValue(t *testing.T) {
	if got := FirstNonEmpty("", "fallback", "later"); got != "fallback" {
		t.Fatalf("FirstNonEmpty = %q, want fallback", got)
	}
	if got := FirstNonEmpty("", ""); got != "" {
		t.Fatalf("FirstNonEmpty empty = %q, want empty", got)
	}
}
