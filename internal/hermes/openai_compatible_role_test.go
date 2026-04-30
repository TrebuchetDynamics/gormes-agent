package hermes

import (
	"encoding/json"
	"testing"
)

func TestOpenAICompatibleDeveloperRoleSerialization(t *testing.T) {
	req := ChatRequest{
		Model: "openai/gpt-5.2",
		Messages: []Message{
			{Role: "system", Content: "You are Gormes."},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "I'll help."},
			{Role: "tool", ToolCallID: "call_1", Name: "read_file", Content: "ok"},
		},
	}
	original, err := json.Marshal(req.Messages)
	if err != nil {
		t.Fatalf("marshal original messages: %v", err)
	}

	body := openAICompatibleRoleRequestBody(t, req)
	messages := decodedMessages(t, body)
	if got := messages[0]["role"]; got != "developer" {
		t.Fatalf("first system role serialized as %q, want developer; body=%#v", got, body)
	}
	if got := messages[1]["role"]; got != "user" {
		t.Fatalf("user role serialized as %q, want user", got)
	}
	if got := messages[2]["role"]; got != "assistant" {
		t.Fatalf("assistant role serialized as %q, want assistant", got)
	}
	if got := messages[3]["role"]; got != "tool" {
		t.Fatalf("tool role serialized as %q, want tool", got)
	}

	after, err := json.Marshal(req.Messages)
	if err != nil {
		t.Fatalf("marshal after messages: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("ChatRequest messages mutated\n before: %s\n  after: %s", original, after)
	}
}

func TestOpenAICompatibleSystemRoleKeptForNonDeveloperModels(t *testing.T) {
	body := openAICompatibleRoleRequestBody(t, ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: "system", Content: "You are Gormes."}, {Role: "user", Content: "hello"}},
	})
	messages := decodedMessages(t, body)
	if got := messages[0]["role"]; got != "system" {
		t.Fatalf("first role serialized as %q, want system for non-developer model", got)
	}
}

func TestOpenAICompatibleDeveloperRoleOnlySwapsFirstSystemMessage(t *testing.T) {
	body := openAICompatibleRoleRequestBody(t, ChatRequest{
		Model: "codex-mini-latest",
		Messages: []Message{
			{Role: "user", Content: "prelude"},
			{Role: "system", Content: "late system note"},
		},
	})
	messages := decodedMessages(t, body)
	if got := messages[0]["role"]; got != "user" {
		t.Fatalf("first role serialized as %q, want user", got)
	}
	if got := messages[1]["role"]; got != "system" {
		t.Fatalf("late system role serialized as %q, want system", got)
	}
}

func TestCodexResponsesKeepsSystemGuidanceAsInstructions(t *testing.T) {
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model:    "codex-mini-latest",
		Messages: []Message{{Role: "system", Content: "You are Gormes."}, {Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload() error = %v", err)
	}
	if payload.Instructions != "You are Gormes." {
		t.Fatalf("Instructions = %q, want system guidance content", payload.Instructions)
	}
	for _, item := range payload.Input {
		if msg, ok := item.(codexResponsesMessageItem); ok && msg.Role == "developer" {
			t.Fatalf("Codex Responses input unexpectedly used developer role: %#v", payload.Input)
		}
	}
}

func openAICompatibleRoleRequestBody(t *testing.T, req ChatRequest) map[string]any {
	t.Helper()
	client := &httpClient{baseURL: "http://example.test", provider: "openai_compatible"}
	body, _, err := client.buildOpenAICompatibleChatRequestBody(req)
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", body, err)
	}
	return decoded
}

func decodedMessages(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, ok := body["messages"].([]any)
	if !ok {
		t.Fatalf("messages missing or not a list: %#v", body)
	}
	messages := make([]map[string]any, 0, len(raw))
	for i, item := range raw {
		msg, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("messages[%d] is %T, want object", i, item)
		}
		messages = append(messages, msg)
	}
	return messages
}
