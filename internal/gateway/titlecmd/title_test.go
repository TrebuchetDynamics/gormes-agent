package titlecmd

import (
	"strings"
	"testing"
)

func TestParseArgHandlesSlashAndRawTitle(t *testing.T) {
	if got, ok := ParseArg("/title   Friendly Greeting  "); !ok || got != "Friendly Greeting" {
		t.Fatalf("ParseArg slash = %q, %v; want Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("/title@GormesBot   Friendly Greeting  "); !ok || got != "Friendly Greeting" {
		t.Fatalf("ParseArg bot mention = %q, %v; want Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("Friendly Greeting"); !ok || got != "Friendly Greeting" {
		t.Fatalf("ParseArg raw = %q, %v; want Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("/title"); ok || got != "" {
		t.Fatalf("ParseArg empty = %q, %v; want empty,false", got, ok)
	}
}

func TestSanitizeCleansControlAndFormattingRunes(t *testing.T) {
	got, err := Sanitize(" Hello\x00\u200b   world\n ")
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	if got != "Hello world" {
		t.Fatalf("Sanitize = %q, want Hello world", got)
	}
}

func TestSanitizeRejectsLongTitles(t *testing.T) {
	_, err := Sanitize(strings.Repeat("x", MaxSessionTitleRunes+1))
	if err == nil {
		t.Fatal("Sanitize long title err = nil, want error")
	}
}
