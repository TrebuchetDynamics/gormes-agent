package textlimit

import "testing"

func TestTruncateMarkdownV2SafeRemovesDanglingEscapeWithoutTruncation(t *testing.T) {
	got := TruncateMarkdownV2Safe(`abc\`, 4)
	want := `abc`
	if got != want {
		t.Fatalf("TruncateMarkdownV2Safe() = %q, want dangling escape removed as %q", got, want)
	}
	if hasDanglingMarkdownEscape(got) {
		t.Fatalf("TruncateMarkdownV2Safe() returned dangling escape: %q", got)
	}
}

func TestTruncateMarkdownV2SafeKeepsCompleteEscapePairAtBoundary(t *testing.T) {
	got := TruncateMarkdownV2Safe(`abc\\def`, 6)
	want := `abc\\…`
	if got != want {
		t.Fatalf("TruncateMarkdownV2Safe() = %q, want complete escape pair %q", got, want)
	}
	if hasDanglingMarkdownEscape(got) {
		t.Fatalf("TruncateMarkdownV2Safe() returned dangling escape: %q", got)
	}
}

func TestTruncateMarkdownV2SafeMovesIncompleteEscapePairBeforeEllipsis(t *testing.T) {
	got := TruncateMarkdownV2Safe(`abc\_def`, 5)
	want := `abc…`
	if got != want {
		t.Fatalf("TruncateMarkdownV2Safe() = %q, want incomplete escape pair removed as %q", got, want)
	}
	if hasDanglingMarkdownEscape(got) {
		t.Fatalf("TruncateMarkdownV2Safe() returned dangling escape: %q", got)
	}
}

func TestTruncateMarkdownV2SafeBoundsTinyLimits(t *testing.T) {
	if got := TruncateMarkdownV2Safe(`\_x`, 1); got != "…" {
		t.Fatalf("TruncateMarkdownV2Safe(max=1) = %q, want ellipsis only", got)
	}
	if got := TruncateMarkdownV2Safe(`\_x`, 2); got != "…" {
		t.Fatalf("TruncateMarkdownV2Safe(max=2) = %q, want incomplete escape removed", got)
	}
}

func hasDanglingMarkdownEscape(s string) bool {
	runes := []rune(s)
	count := 0
	for i := len(runes) - 1; i >= 0 && runes[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}
