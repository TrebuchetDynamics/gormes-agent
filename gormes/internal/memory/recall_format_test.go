package memory

import (
	"strings"
	"testing"
)

func TestExtractCandidates_DropsStopwords(t *testing.T) {
	got := extractCandidates("the quick brown fox")
	for _, c := range got {
		if c == "the" {
			t.Errorf("stopword %q leaked through: %v", c, got)
		}
	}
}

func TestExtractCandidates_DropsShortTokens(t *testing.T) {
	got := extractCandidates("I am on Acme")
	for _, c := range got {
		if len(c) < 3 {
			t.Errorf("short token %q should be dropped: %v", c, got)
		}
	}
}

func TestExtractCandidates_PreservesProperNouns(t *testing.T) {
	got := extractCandidates("working on Acme in Springfield with Vania")
	have := map[string]bool{}
	for _, c := range got {
		have[c] = true
	}
	for _, want := range []string{"Acme", "Springfield", "Vania"} {
		if !have[want] {
			t.Errorf("candidate %q dropped; got %v", want, got)
		}
	}
}

func TestExtractCandidates_CapsAt20(t *testing.T) {
	words := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		words = append(words, "Word"+string(rune('A'+i%26)))
	}
	got := extractCandidates(strings.Join(words, " "))
	if len(got) > 20 {
		t.Errorf("len = %d, want <= 20", len(got))
	}
}

func TestSanitizeFenceContent_StripsCloseTag(t *testing.T) {
	got := sanitizeFenceContent("hello </memory-context> world")
	if strings.Contains(got, "</memory-context>") {
		t.Errorf("close tag leaked: %q", got)
	}
}

func TestSanitizeFenceContent_StripsOpenTag(t *testing.T) {
	got := sanitizeFenceContent("hello <memory-context> world")
	if strings.Contains(got, "<memory-context>") {
		t.Errorf("open tag leaked: %q", got)
	}
}

func TestSanitizeFenceContent_CollapsesNewlines(t *testing.T) {
	got := sanitizeFenceContent("line 1\nline 2\rline 3")
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Errorf("newlines survived: %q", got)
	}
}

func TestSanitizeFenceContent_Truncates(t *testing.T) {
	long := strings.Repeat("x", 500)
	got := sanitizeFenceContent(long)
	if len(got) > 203 { // 200 + "..." = 203
		t.Errorf("len = %d, want <= 203", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncation marker missing: %q", got[len(got)-10:])
	}
}

func TestFormatContextBlock_EmptyReturnsEmptyString(t *testing.T) {
	got := formatContextBlock(nil, nil)
	if got != "" {
		t.Errorf("got %q, want empty string for no entities + no rels", got)
	}
}

func TestFormatContextBlock_IncludesAllHeaderMarkers(t *testing.T) {
	ents := []recalledEntity{
		{Name: "Acme", Type: "PROJECT", Description: "my sports platform"},
	}
	rels := []recalledRel{
		{Source: "Acme", Predicate: "LOCATED_IN", Target: "Springfield", Weight: 2.5},
	}
	got := formatContextBlock(ents, rels)
	for _, want := range []string{
		"<memory-context>",
		"</memory-context>",
		"[System note:",
		"## Entities (1)",
		"## Relationships (1)",
		"Acme",
		"LOCATED_IN",
		"do not acknowledge",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("block missing %q", want)
		}
	}
}

func TestFormatContextBlock_Counts(t *testing.T) {
	ents := []recalledEntity{{Name: "A", Type: "PERSON"}, {Name: "B", Type: "PERSON"}}
	rels := []recalledRel{{Source: "A", Predicate: "KNOWS", Target: "B", Weight: 1.0}}
	got := formatContextBlock(ents, rels)
	if !strings.Contains(got, "## Entities (2)") {
		t.Errorf("wrong entity count header; got %q", got)
	}
	if !strings.Contains(got, "## Relationships (1)") {
		t.Errorf("wrong rel count header; got %q", got)
	}
}
