package cron

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// These helpers cap by byte budget; the fix only requires they never split a
// multibyte UTF-8 sequence (the kept prefix stays valid UTF-8).

func TestTruncateCutsOnRuneBoundary(t *testing.T) {
	s := strings.Repeat("é", 5) // 5 runes, 10 bytes
	got := truncate(s, 3)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
	kept := strings.TrimSuffix(got, "…")
	if !strings.HasPrefix(s, kept) {
		t.Fatalf("truncate kept %q is not a prefix of %q", kept, s)
	}
}

func TestOneLineCutsOnRuneBoundary(t *testing.T) {
	s := strings.Repeat("世", 5) // 5 runes, 15 bytes
	got := oneLine(s, 7)
	if !utf8.ValidString(got) {
		t.Fatalf("oneLine produced invalid UTF-8: %q", got)
	}
	kept := strings.TrimSuffix(got, "…")
	if !strings.HasPrefix(s, kept) {
		t.Fatalf("oneLine kept %q is not a prefix of %q", kept, s)
	}
}

func TestBoundContextFromOutputCutsOnRuneBoundary(t *testing.T) {
	// One rune over the byte budget so the naive cut would split a 3-byte rune.
	s := strings.Repeat("世", maxContextFromOutputChars+10)
	got := boundContextFromOutput(s)
	if !utf8.ValidString(got) {
		t.Fatalf("boundContextFromOutput produced invalid UTF-8")
	}
}
