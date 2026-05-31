package pagination

import (
	"strings"
	"testing"
)

func TestSplitOutboundTextDoesNotCreateDanglingMarkdownEscape(t *testing.T) {
	chunks := splitOutboundText(strings.Repeat("\\", 7)+"done", 4)
	if len(chunks) < 2 {
		t.Fatalf("splitOutboundText() len = %d, want multiple chunks", len(chunks))
	}
	for i, chunk := range chunks[:len(chunks)-1] {
		if hasDanglingMarkdownEscape(chunk) {
			t.Fatalf("splitOutboundText() chunk %d ends with dangling escape: %q; chunks=%q", i, chunk, chunks)
		}
	}
}

func TestSplitOutboundTextKeepsEscapedPairTogetherAtBoundary(t *testing.T) {
	chunks := splitOutboundText("abc\\_def", 4)
	if len(chunks) < 2 {
		t.Fatalf("splitOutboundText() len = %d, want multiple chunks", len(chunks))
	}
	if got := chunks[0]; got != "abc" {
		t.Fatalf("splitOutboundText() first chunk = %q, want escape pair moved to next chunk", got)
	}
	if hasDanglingMarkdownEscape(chunks[0]) {
		t.Fatalf("splitOutboundText() first chunk ends with dangling escape: %q", chunks[0])
	}
}

func hasDanglingMarkdownEscape(s string) bool {
	count := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}
