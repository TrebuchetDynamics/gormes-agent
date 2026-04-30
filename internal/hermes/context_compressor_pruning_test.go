package hermes

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

func containsEvidence(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
