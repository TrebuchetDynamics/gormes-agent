package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestContextCompressorPruning_PrunesOldToolResultsAndKeepsRecentTail(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "initial request"},
		{Role: "assistant", Content: "I will inspect", ToolCalls: []ToolCall{{ID: "old-call", Name: "read_file", Arguments: json.RawMessage(`{"path":"/tmp/very-long.txt","offset":1}`)}}},
		{Role: "tool", ToolCallID: "old-call", Name: "read_file", Content: strings.Repeat("old tool output ", 40)},
		{Role: "assistant", Content: "old answer"},
		{Role: "user", Content: "older followup"},
		{Role: "user", Content: "recent question"},
		{Role: "assistant", Content: "recent answer"},
	}

	got, status := PruneContextMessages(messages, ContextPruningConfig{
		ProtectFirstN:        2,
		TailTokenBudget:      8,
		MinTailMessages:      3,
		ToolResultPruneChars: 120,
		SummaryText:          ContextPruningSummaryPrefix + "\ncompressed middle",
	})

	if status.State != ContextPruningStateReady {
		t.Fatalf("state=%q evidence=%v", status.State, status.Evidence)
	}
	if status.PrunedToolResults != 1 {
		t.Fatalf("pruned=%d want 1", status.PrunedToolResults)
	}
	if len(got) != 6 { // 2 head + summary + 3 tail
		t.Fatalf("len=%d want 6: %#v", len(got), got)
	}
	if got[0].Role != "system" || !strings.Contains(got[0].Content, "compacted into a handoff summary") {
		t.Fatalf("system note missing: %#v", got[0])
	}
	if got[2].Role == got[1].Role || got[2].Role == got[3].Role {
		t.Fatalf("summary role collided with neighbors: %q/%q/%q", got[1].Role, got[2].Role, got[3].Role)
	}
	if !strings.HasPrefix(got[2].Content, ContextPruningSummaryPrefix) {
		t.Fatalf("summary prefix missing: %q", got[2].Content)
	}
	if got[3].Role != "user" || got[4].Role != "user" || got[5].Role != "assistant" {
		t.Fatalf("tail not preserved as last three messages: %#v", got[3:])
	}
}

func TestContextCompressorPruning_SummaryPrefixMatchesCurrentHermesHandoff(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "initial"},
		{Role: "assistant", Content: "old answer"},
		{Role: "user", Content: "middle"},
		{Role: "assistant", Content: "middle answer"},
		{Role: "user", Content: "latest task"},
		{Role: "assistant", Content: "latest answer"},
	}

	got, _ := PruneContextMessages(messages, ContextPruningConfig{
		ProtectFirstN:   2,
		TailTokenBudget: 1,
		MinTailMessages: 2,
		SummaryText:     "## Active Task\nSummarized earlier work.",
	})
	summary := findContextSummaryMessage(t, got)
	if strings.Contains(summary.Content, "resume exactly") {
		t.Fatalf("summary prefix kept stale resume-exactly directive: %q", summary.Content)
	}
	if !strings.Contains(summary.Content, "latest message WINS") {
		t.Fatalf("summary prefix missing current latest-message-wins guard: %q", summary.Content)
	}
	if !strings.Contains(summary.Content, "persistent memory") {
		t.Fatalf("summary prefix missing current persistent-memory guard: %q", summary.Content)
	}
}

func TestContextCompressorPruning_NormalizesLegacySummaryPrefixes(t *testing.T) {
	normalized := NormalizeContextPruningSummary("[CONTEXT SUMMARY]:\nlegacy body")
	if !strings.HasPrefix(normalized, ContextPruningSummaryPrefix) {
		t.Fatalf("normalized summary missing current prefix: %q", normalized)
	}
	if strings.Contains(normalized, "[CONTEXT SUMMARY]") {
		t.Fatalf("normalized summary retained legacy prefix: %q", normalized)
	}
	if strings.Count(normalized, "legacy body") != 1 {
		t.Fatalf("normalized summary body count = %d, want 1: %q", strings.Count(normalized, "legacy body"), normalized)
	}

	historical := NormalizeContextPruningSummary(contextPruningHistoricalSummaryPrefixResumeExactly + "\nhistorical body")
	if strings.Contains(historical, "resume exactly") {
		t.Fatalf("historical prefix retained stale resume-exactly directive: %q", historical)
	}
	if strings.Count(historical, "historical body") != 1 {
		t.Fatalf("historical body count = %d, want 1: %q", strings.Count(historical, "historical body"), historical)
	}
}

func TestContextCompressorPruning_RehydratesPreviousSummaryFromHandoff(t *testing.T) {
	oldSummary := "RESUMED-SUMMARY-BODY durable continuity facts"
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: ContextPruningSummaryPrefix + "\n" + oldSummary},
		{Role: "assistant", Content: "handoff acknowledged after resume"},
		{Role: "user", Content: "new user turn after resume"},
		{Role: "assistant", Content: "new assistant work after resume"},
		{Role: "user", Content: "more new work after resume"},
		{Role: "assistant", Content: "latest tail response"},
		{Role: "user", Content: "final active request stays in protected tail"},
	}

	plan := PlanContextSummaryLineage(messages, 2, 7, "")
	if plan.SummaryIndex != 1 {
		t.Fatalf("SummaryIndex = %d, want resumed handoff at index 1", plan.SummaryIndex)
	}
	if plan.PreviousSummary != oldSummary {
		t.Fatalf("PreviousSummary = %q, want %q", plan.PreviousSummary, oldSummary)
	}
	if plan.TurnsStart != 2 || plan.TurnsEnd != 7 {
		t.Fatalf("turn window = [%d:%d], want [2:7]", plan.TurnsStart, plan.TurnsEnd)
	}
	if len(plan.TurnsToSummarize) == 0 {
		t.Fatal("TurnsToSummarize empty, want new resumed turns")
	}
	for _, msg := range plan.TurnsToSummarize {
		if strings.Contains(msg.Content, oldSummary) || strings.HasPrefix(msg.Content, ContextPruningSummaryPrefix) {
			t.Fatalf("handoff summary was serialized as a new turn: %+v", msg)
		}
	}
}

func TestContextCompressorPruning_ExistingPreviousSummaryExcludesPersistedHandoff(t *testing.T) {
	oldSummary := "OLD-SUMMARY-BODY unique continuity facts"
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: ContextPruningSummaryPrefix + "\n" + oldSummary},
		{Role: "assistant", Content: "handoff acknowledged after resume"},
		{Role: "user", Content: "new user turn after resume"},
		{Role: "assistant", Content: "new assistant work after resume"},
		{Role: "user", Content: "more new work after resume"},
		{Role: "assistant", Content: "latest tail response"},
		{Role: "user", Content: "final active request stays in protected tail"},
	}

	plan := PlanContextSummaryLineage(messages, 2, 7, oldSummary)
	if plan.PreviousSummary != oldSummary {
		t.Fatalf("PreviousSummary = %q, want existing summary preserved", plan.PreviousSummary)
	}
	if plan.Rehydrated {
		t.Fatal("Rehydrated = true, want false when previous summary already exists")
	}
	for _, msg := range plan.TurnsToSummarize {
		if strings.Contains(msg.Content, oldSummary) || strings.HasPrefix(msg.Content, ContextPruningSummaryPrefix) {
			t.Fatalf("existing handoff summary was serialized as a new turn: %+v", msg)
		}
	}
}

func TestContextCompressorPruning_MultimodalContentPartsUseFlatImageBudget(t *testing.T) {
	text := strings.Repeat("x", 400)
	msg := Message{
		Role:    "user",
		Content: text,
		ContentParts: []MessageContentPart{
			{Type: "text", Text: text},
			{Type: "image_url", ImageURL: "data:image/png;base64," + strings.Repeat("A", 1_000_000)},
		},
	}

	want := (len(text)+imageCharEquivalent)/charsPerToken + 10
	if got := estimatePruningMessageTokens(msg); got != want {
		t.Fatalf("estimatePruningMessageTokens(multimodal) = %d, want %d (text once + flat image charge)", got, want)
	}
}

func TestContextCompressorPruning_DoesNotSplitToolCallPairsAtTail(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "initial"},
		{Role: "assistant", Content: "old"},
		{Role: "user", Content: "please run"},
		{Role: "assistant", Content: "running", ToolCalls: []ToolCall{{ID: "tail-call", Name: "terminal", Arguments: json.RawMessage(`{"command":"go test ./..."}`)}}},
		{Role: "tool", ToolCallID: "tail-call", Name: "terminal", Content: "ok"},
	}

	got, status := PruneContextMessages(messages, ContextPruningConfig{
		ProtectFirstN:   2,
		TailTokenBudget: 1,
		MinTailMessages: 1,
	})
	if status.State != ContextPruningStateReady {
		t.Fatalf("state=%q evidence=%v", status.State, status.Evidence)
	}
	foundAssistant, foundTool := false, false
	for _, msg := range got {
		if msg.Role == "assistant" && len(msg.ToolCalls) == 1 && msg.ToolCalls[0].ID == "tail-call" {
			foundAssistant = true
		}
		if msg.Role == "tool" && msg.ToolCallID == "tail-call" {
			foundTool = true
		}
	}
	if !foundAssistant || !foundTool {
		t.Fatalf("tail pair split assistant=%v tool=%v messages=%#v", foundAssistant, foundTool, got)
	}
	if got[len(got)-1].Role != "tool" || got[len(got)-2].Role != "assistant" {
		t.Fatalf("tool pair not adjacent at tail: %#v", got)
	}
}

func TestContextCompressorPruning_InvalidToolCallArgumentsDegradeWithoutMutation(t *testing.T) {
	badArgs := json.RawMessage(`{"path":"/tmp","content":"unterminated`)
	messages := []Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "initial"},
		{Role: "assistant", Content: "write", ToolCalls: []ToolCall{{ID: "bad", Name: "write_file", Arguments: badArgs}}},
		{Role: "tool", ToolCallID: "bad", Name: "write_file", Content: strings.Repeat("result ", 40)},
		{Role: "user", Content: "latest"},
		{Role: "assistant", Content: "ok"},
	}

	got, status := PruneContextMessages(messages, ContextPruningConfig{
		ProtectFirstN:        2,
		TailTokenBudget:      1,
		MinTailMessages:      2,
		ToolResultPruneChars: 50,
	})
	if status.State != ContextPruningStateDegraded {
		t.Fatalf("state=%q evidence=%v", status.State, status.Evidence)
	}
	if !containsEvidence(status.Evidence, ContextPruningEvidenceInvalidToolPair) {
		t.Fatalf("missing invalid-tool evidence: %v", status.Evidence)
	}
	for _, msg := range got {
		for _, tc := range msg.ToolCalls {
			if tc.ID == "bad" && string(tc.Arguments) != string(badArgs) {
				t.Fatalf("invalid arguments mutated: %q", string(tc.Arguments))
			}
		}
	}
}

func findContextSummaryMessage(t *testing.T, messages []Message) Message {
	t.Helper()
	for _, msg := range messages {
		if strings.HasPrefix(msg.Content, ContextPruningSummaryPrefix) {
			return msg
		}
	}
	t.Fatalf("context summary message not found in %#v", messages)
	return Message{}
}

func containsEvidence(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
