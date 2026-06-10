package pagination

import (
	"strings"
	"testing"
)

func TestTelegramTextDoesNotTreatUnpaginatedValidMarkerTextAsPageMarker(t *testing.T) {
	input := "intro\\\n\n\\(1/2\\)"
	pages := TelegramText(input)
	if len(pages) != 1 {
		t.Fatalf("TelegramText() pages = %q, want one page", pages)
	}
	if got := pages[0]; got != input {
		t.Fatalf("TelegramText() = %q, want unpaginated marker-like body preserved as %q", got, input)
	}
}

func TestTelegramTextDoesNotMistakeBodyTextForPageMarker(t *testing.T) {
	pages := TelegramText("intro\n\n\\(not-a-page-marker\\")
	if len(pages) != 1 {
		t.Fatalf("TelegramText() pages = %q, want one page", pages)
	}
	if hasDanglingMarkdownEscape(pages[0]) {
		t.Fatalf("TelegramText() kept dangling escape after marker-like body text: %q", pages[0])
	}
	if got, want := pages[0], "intro\n\n\\(not-a-page-marker"; got != want {
		t.Fatalf("TelegramText() = %q, want %q", got, want)
	}
}

func TestTelegramTextRemovesDanglingEscapeBeforePageMarker(t *testing.T) {
	input := strings.Repeat("a", MaxMessageLen) + "\\"
	pages := TelegramText(input)
	if len(pages) < 2 {
		t.Fatalf("TelegramText() pages = %d, want multiple pages", len(pages))
	}
	last := pages[len(pages)-1]
	marker := "\n\n\\("
	body, _, ok := strings.Cut(last, marker)
	if !ok {
		t.Fatalf("last page %q missing telegram page marker", last)
	}
	if hasDanglingMarkdownEscape(body) {
		t.Fatalf("TelegramText() last page body ends with dangling escape before marker: %q", last)
	}
}

func TestTelegramTextRemovesDanglingEscapeWithoutPagination(t *testing.T) {
	pages := TelegramText(`abc\`)
	if len(pages) != 1 {
		t.Fatalf("TelegramText() pages = %q, want one page", pages)
	}
	if got, want := pages[0], `abc`; got != want {
		t.Fatalf("TelegramText() = %q, want dangling escape removed as %q", got, want)
	}
	if hasDanglingMarkdownEscape(pages[0]) {
		t.Fatalf("TelegramText() returned dangling escape: %q", pages[0])
	}
}

func TestSplitOutboundTextDoesNotCreateDanglingEscapeAtTinyLimit(t *testing.T) {
	chunks := splitOutboundText(`\_x`, 1)
	if len(chunks) < 2 {
		t.Fatalf("splitOutboundText() len = %d, want multiple chunks", len(chunks))
	}
	for i, chunk := range chunks[:len(chunks)-1] {
		if hasDanglingMarkdownEscape(chunk) {
			t.Fatalf("splitOutboundText() chunk %d ends with dangling escape: %q; chunks=%q", i, chunk, chunks)
		}
	}
}

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
