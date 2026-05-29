package llm

import (
	"encoding/json"
	"testing"
)

func TestCodexResponsesMultimodalToolResultArray(t *testing.T) {
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model: "gpt-5-codex",
		Messages: []Message{
			{Role: "user", Content: "what is in /tmp/x.png?"},
			{
				Role: "assistant",
				ToolCalls: []ToolCall{{
					ID:        "call_vision",
					Name:      "vision_analyze",
					Arguments: json.RawMessage(`{"image_url":"/tmp/x.png"}`),
				}},
			},
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
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload: %v", err)
	}

	item := codexToolOutputItem(t, payload.Input)
	output, ok := item["output"].([]any)
	if !ok {
		t.Fatalf("function_call_output.output = %T, want array: %#v", item["output"], item["output"])
	}
	if len(output) != 2 {
		t.Fatalf("output len = %d, want 2: %#v", len(output), output)
	}
	textPart := output[0].(map[string]any)
	if textPart["type"] != "input_text" || textPart["text"] != "Image loaded." {
		t.Fatalf("text part = %#v", textPart)
	}
	imagePart := output[1].(map[string]any)
	if imagePart["type"] != "input_image" || imagePart["image_url"] != "data:image/png;base64,AAA" || imagePart["detail"] != "high" {
		t.Fatalf("image part = %#v", imagePart)
	}
}

func TestCodexResponsesStringToolResultUnchanged(t *testing.T) {
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model: "gpt-5-codex",
		Messages: []Message{
			{Role: "assistant", ToolCalls: []ToolCall{{ID: "call_echo", Name: "echo", Arguments: json.RawMessage(`{}`)}}},
			{Role: "tool", ToolCallID: "call_echo", Name: "echo", Content: "plain text output"},
		},
	})
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload: %v", err)
	}
	item := codexToolOutputItem(t, payload.Input)
	if item["output"] != "plain text output" {
		t.Fatalf("output = %#v, want string output unchanged", item["output"])
	}
}

func codexToolOutputItem(t *testing.T, input []any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("unmarshal input %s: %v", raw, err)
	}
	for _, item := range items {
		if item["type"] == "function_call_output" {
			return item
		}
	}
	t.Fatalf("no function_call_output in %#v", items)
	return nil
}
