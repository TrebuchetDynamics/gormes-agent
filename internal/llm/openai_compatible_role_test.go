package llm

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

func TestOpenAICompatibleToolResultContentParts(t *testing.T) {
	body := openAICompatibleRoleRequestBody(t, ChatRequest{
		Model: "openai/gpt-5.4",
		Messages: []Message{
			{
				Role:       "tool",
				ToolCallID: "call_vision",
				Name:       "vision_analyze",
				Content:    "Image attached natively for the main model.",
				ContentParts: []MessageContentPart{
					{Type: "text", Text: "Image loaded."},
					{Type: "image_url", ImageURL: "data:image/png;base64,AAA", Detail: "high"},
				},
			},
		},
	})
	messages := decodedMessages(t, body)
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	msg := messages[0]
	if msg["role"] != "tool" || msg["tool_call_id"] != "call_vision" || msg["name"] != "vision_analyze" {
		t.Fatalf("tool metadata = %#v", msg)
	}
	content, ok := msg["content"].([]any)
	if !ok {
		t.Fatalf("content = %T, want array: %#v", msg["content"], msg["content"])
	}
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2: %#v", len(content), content)
	}
	text := content[0].(map[string]any)
	if text["type"] != "text" || text["text"] != "Image loaded." {
		t.Fatalf("text part = %#v", text)
	}
	image := content[1].(map[string]any)
	imageURL, _ := image["image_url"].(map[string]any)
	if image["type"] != "image_url" || imageURL["url"] != "data:image/png;base64,AAA" || imageURL["detail"] != "high" {
		t.Fatalf("image part = %#v", image)
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
