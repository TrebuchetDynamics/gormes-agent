package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSummarizePrunedSkillViewAddsReloadMarker(t *testing.T) {
	got := summarizePrunedToolResult("skill_view", json.RawMessage(`{"name":"demo-skill"}`), strings.Repeat("x", 200))
	if !strings.HasPrefix(got, "[skill_view] name=demo-skill") {
		t.Fatalf("summary = %q", got)
	}
	if !strings.Contains(got, SkillPrunedMarkerPrefix) || !strings.Contains(got, "skill_view(name='demo-skill')") {
		t.Fatalf("summary missing skill marker: %q", got)
	}
}

func TestPruneContextMessagesAddsSkillSafetyRuleOnce(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "load the demo skill"},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Name: "skill_view", Arguments: json.RawMessage(`{"name":"demo-skill"}`)}}},
		{Role: "tool", ToolCallID: "c1", Name: "skill_view", Content: strings.Repeat("x", 600)},
		{Role: "user", Content: "continue"},
		{Role: "assistant", Content: "working"},
		{Role: "user", Content: "latest request"},
	}
	compressed, status := PruneContextMessages(messages, ContextPruningConfig{
		TailTokenBudget:      1,
		MinTailMessages:      1,
		ToolResultPruneChars: 10,
		SummaryText:          "summary body",
	})
	if status.PrunedToolResults == 0 {
		t.Fatalf("status = %+v, want one pruned tool result", status)
	}
	joined := ""
	for _, msg := range compressed {
		joined += msg.Content + "\n"
	}
	if !strings.Contains(joined, SkillPrunedMarkerPrefix) {
		t.Fatalf("compressed transcript missing skill marker:\n%s", joined)
	}
	if got := strings.Count(joined, SkillSafetyRule); got != 1 {
		t.Fatalf("Skill Safety Rule count = %d, want 1:\n%s", got, joined)
	}
	if !strings.Contains(joined, "pattern='demo-skill'") && !strings.Contains(joined, "name='demo-skill'") {
		t.Fatalf("compressed transcript missing reload pointer:\n%s", joined)
	}
}

func TestPruneContextMessagesOmitsSkillSafetyRuleWithoutMarker(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
		{Role: "user", Content: "latest request"},
	}
	compressed, _ := PruneContextMessages(messages, ContextPruningConfig{
		TailTokenBudget:      1,
		MinTailMessages:      1,
		ToolResultPruneChars: 10,
		SummaryText:          "summary body",
	})
	for _, msg := range compressed {
		if strings.Contains(msg.Content, SkillSafetyRule) {
			t.Fatalf("unexpected Skill Safety Rule without a prune marker:\n%s", msg.Content)
		}
	}
}

func TestWithoutSkillSafetyRuleRemovesExistingCopy(t *testing.T) {
	body := "summary body\n\n" + SkillSafetyRule
	if got := withoutSkillSafetyRule(body); strings.Contains(got, SkillSafetyRule) {
		t.Fatalf("rule not removed: %q", got)
	}
}
