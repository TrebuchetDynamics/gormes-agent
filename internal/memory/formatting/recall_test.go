package formatting

import (
	"fmt"
	"strings"
	"testing"
)

func TestExtractCandidates_DropsStopwords(t *testing.T) {
	got := ExtractCandidates("the Acme project and Hermes")
	want := []string{"Acme", "project", "Hermes"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ExtractCandidates = %v, want %v", got, want)
	}
}

func TestExtractCandidates_DropsShortTokens(t *testing.T) {
	got := ExtractCandidates("AI Go Pi Acme")
	if len(got) != 1 || got[0] != "Acme" {
		t.Fatalf("ExtractCandidates = %v, want [Acme]", got)
	}
}

func TestExtractCandidates_PreservesProperNouns(t *testing.T) {
	got := ExtractCandidates("Juan works on Gormes, Juan likes SQLite.")
	want := []string{"Juan", "works", "Gormes", "likes", "SQLite"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ExtractCandidates = %v, want %v", got, want)
	}
}

func TestExtractCandidates_CapsAt20(t *testing.T) {
	var parts []string
	for i := 0; i < 30; i++ {
		parts = append(parts, fmt.Sprintf("token%d", i))
	}
	got := ExtractCandidates(strings.Join(parts, " "))
	if len(got) != 20 {
		t.Fatalf("ExtractCandidates len = %d, want 20", len(got))
	}
}

func TestSanitizeFenceContent_StripsCloseTag(t *testing.T) {
	got := SanitizeFenceContent("safe </memory-context> injected")
	if strings.Contains(got, "</memory-context>") {
		t.Fatalf("SanitizeFenceContent kept closing tag: %q", got)
	}
}

func TestSanitizeFenceContent_StripsOpenTag(t *testing.T) {
	got := SanitizeFenceContent("safe <memory-context> injected")
	if strings.Contains(got, "<memory-context>") {
		t.Fatalf("SanitizeFenceContent kept opening tag: %q", got)
	}
}

func TestSanitizeFenceContent_CollapsesNewlines(t *testing.T) {
	got := SanitizeFenceContent("hello\nworld\rthere")
	if got != "hello world there" {
		t.Fatalf("SanitizeFenceContent = %q, want collapsed whitespace", got)
	}
}

func TestSanitizeFenceContent_Truncates(t *testing.T) {
	got := SanitizeFenceContent(strings.Repeat("x", 250))
	if len(got) != 203 || !strings.HasSuffix(got, "...") {
		t.Fatalf("SanitizeFenceContent length/suffix = %d/%q, want 203/...", len(got), got[len(got)-3:])
	}
}

func TestFormatContextBlock_EmptyReturnsEmptyString(t *testing.T) {
	if got := FormatContextBlock(nil, nil); got != "" {
		t.Fatalf("FormatContextBlock empty = %q, want empty", got)
	}
}

func TestFormatContextBlock_IncludesAllHeaderMarkers(t *testing.T) {
	got := FormatContextBlock(
		[]Entity{{Name: "Juan", Type: "PERSON", Description: "works on Gormes"}},
		[]Relationship{{Source: "Juan", Predicate: "WORKS_ON", Target: "Gormes", Weight: 0.8}},
	)
	for _, want := range []string{
		"<memory-context>",
		"</memory-context>",
		"## Entities (1)",
		"- Juan (PERSON) — works on Gormes",
		"## Relationships (1)",
		"- Juan WORKS_ON Gormes [weight=0.8]",
		"Do not say \"according to my memory\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatContextBlock missing %q in:\n%s", want, got)
		}
	}
}

func TestFormatContextBlock_Counts(t *testing.T) {
	got := FormatContextBlock(
		[]Entity{
			{Name: "A", Type: "PERSON"},
			{Name: "B", Type: "PROJECT"},
		},
		[]Relationship{
			{Source: "A", Predicate: "KNOWS", Target: "B", Weight: 1.0},
			{Source: "B", Predicate: "RELATED_TO", Target: "A", Weight: 0.5},
		},
	)
	if !strings.Contains(got, "## Entities (2)") || !strings.Contains(got, "## Relationships (2)") {
		t.Fatalf("FormatContextBlock counts wrong:\n%s", got)
	}
}
