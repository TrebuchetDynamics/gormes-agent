package composer

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCollapseComposerPasteLabelPreservesUTF8(t *testing.T) {
	text := "a" + strings.Repeat("界", 30) + "\n" + strings.Repeat("emoji🙂", 30)

	got := CollapseComposerPaste(text, ComposerPasteOptions{LargePasteChars: 10, LargePasteLines: 99})
	if got.Snippet == nil {
		t.Fatal("CollapseComposerPaste did not create snippet for large paste")
	}
	if !utf8.ValidString(got.InsertText) {
		t.Fatalf("CollapseComposerPaste label is invalid UTF-8: %q", got.InsertText)
	}
	if got.Snippet.Text != text {
		t.Fatal("CollapseComposerPaste snippet text changed while making label")
	}
}
