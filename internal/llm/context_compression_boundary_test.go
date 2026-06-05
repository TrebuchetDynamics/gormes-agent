package llm

import (
	"encoding/json"
	"testing"
)

func TestContextCompressionBoundaryProtectsSystemPlusFirstNonSystemMessages(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "first user"},
		{Role: "assistant", Content: "first assistant"},
		{Role: "user", Content: "second user"},
		{Role: "assistant", Content: "middle assistant"},
		{Role: "user", Content: "latest user"},
	}

	boundary := PlanContextCompressionBoundary(messages, ContextCompressionBoundaryOptions{ProtectFirstN: 3, TailTokenBudget: 8})
	if boundary.HeadEnd != 4 {
		t.Fatalf("HeadEnd = %d, want system + first 3 non-system messages protected", boundary.HeadEnd)
	}
	if boundary.CompressStart != 4 {
		t.Fatalf("CompressStart = %d, want protected head boundary", boundary.CompressStart)
	}
}

func TestContextCompressionBoundaryTailDoesNotSplitToolCallGroups(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "read files"},
		assistantWithToolCalls("tc_A", "tc_B", "tc_C", "tc_D"),
		toolResult("tc_A", "A"),
		toolResult("tc_B", "B"),
		toolResult("tc_C", "C"),
		toolResult("tc_D", "D"),
		{Role: "user", Content: "now summarize them"},
		{Role: "assistant", Content: "summary"},
		{Role: "user", Content: "thanks"},
	}

	boundary := PlanContextCompressionBoundary(messages, ContextCompressionBoundaryOptions{ProtectFirstN: 3, TailTokenBudget: 32})
	if boundary.TailStart != 4 {
		t.Fatalf("TailStart = %d, want boundary pulled back before parent assistant tool-call group", boundary.TailStart)
	}
	assertNoOrphanedToolResults(t, messages[boundary.TailStart:])
}

func TestContextCompressionBoundaryKeepsLatestUserInTail(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "initial"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "current task"},
		{Role: "assistant", Content: "working"},
		assistantWithToolCalls("tc_1"),
		toolResult("tc_1", "large output"),
		{Role: "assistant", Content: "final tool summary"},
	}

	boundary := PlanContextCompressionBoundary(messages, ContextCompressionBoundaryOptions{ProtectFirstN: 1, TailTokenBudget: 1})
	if boundary.TailStart > 3 {
		t.Fatalf("TailStart = %d, want latest user message at index 3 included in protected tail", boundary.TailStart)
	}
}

func assistantWithToolCalls(ids ...string) Message {
	calls := make([]ToolCall, 0, len(ids))
	for _, id := range ids {
		calls = append(calls, ToolCall{ID: id, Name: "read_file", Arguments: json.RawMessage(`{"path":"fixture.txt"}`)})
	}
	return Message{Role: "assistant", ToolCalls: calls}
}

func toolResult(id, content string) Message {
	return Message{Role: "tool", ToolCallID: id, Name: "read_file", Content: content}
}

func assertNoOrphanedToolResults(t *testing.T, messages []Message) {
	t.Helper()
	parents := map[string]bool{}
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, call := range msg.ToolCalls {
			parents[call.ID] = true
		}
	}
	for _, msg := range messages {
		if msg.Role == "tool" && !parents[msg.ToolCallID] {
			t.Fatalf("tool result %q has no parent assistant call in protected tail: %+v", msg.ToolCallID, messages)
		}
	}
}
