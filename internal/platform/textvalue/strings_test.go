package textvalue

import "testing"

func TestFirstNonEmptyTrimmed(t *testing.T) {
	got := FirstNonEmptyTrimmed("", " \t ", "  provider  ", "fallback")
	if got != "provider" {
		t.Fatalf("FirstNonEmptyTrimmed() = %q, want provider", got)
	}
}

func TestFirstNonEmptyTrimmedAllBlank(t *testing.T) {
	if got := FirstNonEmptyTrimmed("", "  "); got != "" {
		t.Fatalf("FirstNonEmptyTrimmed blank = %q, want empty", got)
	}
}

func TestCompactWhitespace(t *testing.T) {
	if got := CompactWhitespace("  gormes\tprovider\nmodel   set  "); got != "gormes provider model set" {
		t.Fatalf("CompactWhitespace() = %q", got)
	}
}
