package memory

import (
	"strings"
	"testing"
)

func TestFormatBatchPrompt_IncludesRolePrefix(t *testing.T) {
	rows := []turnRow{
		{id: 1, role: "user", content: "hello"},
		{id: 2, role: "assistant", content: "hi"},
	}
	got := formatBatchPrompt(rows)
	if !strings.Contains(got, "[user]: hello") {
		t.Errorf("prompt missing [user]: hello; got %q", got)
	}
	if !strings.Contains(got, "[assistant]: hi") {
		t.Errorf("prompt missing [assistant]: hi; got %q", got)
	}
}

func TestFormatBatchPrompt_TruncatesLongContent(t *testing.T) {
	long := strings.Repeat("x", 5000)
	got := formatBatchPrompt([]turnRow{{id: 1, role: "user", content: long}})
	if strings.Count(got, "x") > 4000 {
		t.Errorf("content not truncated to 4000 chars; got %d", strings.Count(got, "x"))
	}
}

func TestExtractorSystemPrompt_MentionsPredicateWhitelist(t *testing.T) {
	for _, pred := range []string{"WORKS_ON", "KNOWS", "RELATED_TO"} {
		if !strings.Contains(extractorSystemPrompt, pred) {
			t.Errorf("system prompt missing predicate %q", pred)
		}
	}
}

func TestExtractorSystemPrompt_MentionsTypeWhitelist(t *testing.T) {
	for _, typ := range []string{"PERSON", "PROJECT", "OTHER"} {
		if !strings.Contains(extractorSystemPrompt, typ) {
			t.Errorf("system prompt missing type %q", typ)
		}
	}
}
