package extraction

import (
	"strings"
	"testing"
)

func TestFormatBatchPrompt_IncludesRolePrefix(t *testing.T) {
	rows := []Turn{
		{ID: 1, Role: "user", Content: "hello"},
		{ID: 2, Role: "assistant", Content: "hi"},
	}
	got := FormatBatchPrompt(rows)
	if !strings.Contains(got, "[user]: hello") {
		t.Errorf("prompt missing [user]: hello; got %q", got)
	}
	if !strings.Contains(got, "[assistant]: hi") {
		t.Errorf("prompt missing [assistant]: hi; got %q", got)
	}
}

func TestFormatBatchPrompt_TruncatesLongContent(t *testing.T) {
	long := strings.Repeat("x", 5000)
	got := FormatBatchPrompt([]Turn{{ID: 1, Role: "user", Content: long}})
	if strings.Count(got, "x") > 4000 {
		t.Errorf("content not truncated to 4000 chars; got %d", strings.Count(got, "x"))
	}
}

func TestSystemPrompt_MentionsPredicateWhitelist(t *testing.T) {
	for _, pred := range []string{"WORKS_ON", "KNOWS", "RELATED_TO"} {
		if !strings.Contains(SystemPrompt, pred) {
			t.Errorf("system prompt missing predicate %q", pred)
		}
	}
}

func TestSystemPrompt_MentionsTypeWhitelist(t *testing.T) {
	for _, typ := range []string{"PERSON", "PROJECT", "OTHER"} {
		if !strings.Contains(SystemPrompt, typ) {
			t.Errorf("system prompt missing type %q", typ)
		}
	}
}
