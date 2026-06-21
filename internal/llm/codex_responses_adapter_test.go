package llm

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildCodexResponsesPayload_ConvertsChatInputToolsAndCallIDs(t *testing.T) {
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model:     "gpt-5-codex",
		MaxTokens: 2048,
		Messages: []Message{
			{Role: "system", Content: "You are Gormes."},
			{Role: "user", Content: "Plain text request."},
			{
				Role: "user",
				ContentParts: []MessageContentPart{
					{Type: "text", Text: "Inspect this screenshot."},
					{Type: "image_url", ImageURL: "data:image/png;base64,abc123", Detail: "high"},
				},
			},
			{
				Role:    "assistant",
				Content: "Checking status.",
				ToolCalls: []ToolCall{{
					Name:      "lookup",
					Arguments: json.RawMessage(`{"query":"status"}`),
				}},
			},
			{Role: "tool", ToolCallID: "call_existing|fc_existing", Content: `{"ok":true}`},
		},
		Tools: []ToolDescriptor{{
			Name:        "lookup",
			Description: "Looks up fixture status.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"],"additionalProperties":false}`),
		}},
	})
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload() error = %v", err)
	}

	got := mustMarshalIndent(t, payload)
	want := []byte(`{
  "model": "gpt-5-codex",
  "instructions": "You are Gormes.",
  "input": [
    {
      "role": "user",
      "content": "Plain text request."
    },
    {
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "Inspect this screenshot."
        },
        {
          "type": "input_image",
          "image_url": "data:image/png;base64,abc123",
          "detail": "high"
        }
      ]
    },
    {
      "role": "assistant",
      "content": "Checking status."
    },
    {
      "type": "function_call",
      "call_id": "call_7685ce46427f",
      "name": "lookup",
      "arguments": "{\"query\":\"status\"}"
    },
    {
      "type": "function_call_output",
      "call_id": "call_existing",
      "output": "{\"ok\":true}"
    }
  ],
  "tools": [
    {
      "type": "function",
      "name": "lookup",
      "description": "Looks up fixture status.",
      "strict": false,
      "parameters": {
        "additionalProperties": false,
        "properties": {
          "query": {
            "type": "string"
          }
        },
        "required": [
          "query"
        ],
        "type": "object"
      }
    }
  ],
  "store": false,
  "max_output_tokens": 2048,
  "tool_choice": "auto",
  "parallel_tool_calls": true
}
`)
	if !bytes.Equal(got, want) {
		t.Fatalf("Codex Responses payload mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildCodexResponsesPayload_SerializesPromptCacheKeyFromSession(t *testing.T) {
	base := ChatRequest{
		Model:     "gpt-5-codex",
		SessionID: "session-cache-key-123",
		Messages: []Message{
			{Role: "system", Content: "Stable system instructions."},
			{Role: "user", Content: "First prompt."},
		},
	}
	first, err := buildCodexResponsesPayload(base)
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload(first) error = %v", err)
	}
	if first.PromptCacheKey != "session-cache-key-123" {
		t.Fatalf("PromptCacheKey = %q, want stable session id", first.PromptCacheKey)
	}

	base.Messages[1].Content = "Different current prompt."
	second, err := buildCodexResponsesPayload(base)
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload(second) error = %v", err)
	}
	if second.PromptCacheKey != first.PromptCacheKey {
		t.Fatalf("PromptCacheKey changed across current prompt: %q vs %q", second.PromptCacheKey, first.PromptCacheKey)
	}

	base.SessionID = ""
	withoutSession, err := buildCodexResponsesPayload(base)
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload(without session) error = %v", err)
	}
	if strings.TrimSpace(withoutSession.PromptCacheKey) != "" {
		t.Fatalf("PromptCacheKey = %q without session id, want omitted", withoutSession.PromptCacheKey)
	}
}

func TestNormalizeCodexResponsesResponse_MapsOutputItemsUsageAndToolCalls(t *testing.T) {
	got, err := normalizeCodexResponsesResponse(codexResponsesResponse{
		Status: "completed",
		Output: []codexResponsesOutputItem{
			{
				Type:             "reasoning",
				ID:               "rs_1",
				EncryptedContent: "enc_opaque",
				Summary: []codexResponsesOutputContent{
					{Type: "summary_text", Text: "Need status lookup."},
				},
			},
			{
				Type:   "message",
				Status: "completed",
				Content: []codexResponsesOutputContent{
					{Type: "output_text", Text: "Checking status."},
				},
			},
			{
				Type:      "function_call",
				ID:        "fc_lookup",
				CallID:    "call_lookup",
				Name:      "lookup",
				Arguments: json.RawMessage(`{"query":"status"}`),
			},
		},
		Usage: codexResponsesUsage{
			InputTokens:  21,
			OutputTokens: 8,
			TotalTokens:  29,
		},
	})
	if err != nil {
		t.Fatalf("normalizeCodexResponsesResponse() error = %v", err)
	}

	if got.Message.Role != "assistant" || got.Message.Content != "Checking status." {
		t.Fatalf("message = %+v, want assistant content", got.Message)
	}
	if got.Message.Reasoning == nil || got.Message.Reasoning.Text != "Need status lookup." {
		t.Fatalf("message reasoning = %+v, want summary text", got.Message.Reasoning)
	}
	if len(got.Message.ToolCalls) != 1 {
		t.Fatalf("message tool calls len = %d, want 1", len(got.Message.ToolCalls))
	}
	call := got.Message.ToolCalls[0]
	if call.ID != "call_lookup" || call.Name != "lookup" || string(call.Arguments) != `{"query":"status"}` {
		t.Fatalf("message tool call = %+v, want lookup call", call)
	}
	if got.Usage.InputTokens != 21 || got.Usage.OutputTokens != 8 || got.Usage.TotalTokens != 29 {
		t.Fatalf("usage = %+v, want 21/8/29", got.Usage)
	}

	if len(got.Events) != 3 {
		t.Fatalf("events len = %d, want reasoning/token/done: %+v", len(got.Events), got.Events)
	}
	if got.Events[0].Kind != EventReasoning || got.Events[0].Reasoning != "Need status lookup." {
		t.Fatalf("event[0] = %+v, want reasoning event", got.Events[0])
	}
	if got.Events[1].Kind != EventToken || got.Events[1].Token != "Checking status." {
		t.Fatalf("event[1] = %+v, want token event", got.Events[1])
	}
	final := got.Events[2]
	if final.Kind != EventDone || final.FinishReason != "tool_calls" || final.TokensIn != 21 || final.TokensOut != 8 {
		t.Fatalf("final event = %+v, want tool_calls with usage", final)
	}
	if len(final.ToolCalls) != 1 || final.ToolCalls[0].ID != "call_lookup" {
		t.Fatalf("final tool calls = %+v, want call_lookup", final.ToolCalls)
	}
}

func TestCodexResponsesToolSchema_StripsPatternAndFormat(t *testing.T) {
	// xAI's /responses endpoint rejects "pattern" and "format" keywords in tool
	// schemas (HTTP 400). Mirrors Hermes fix(schema_sanitizer): strip
	// pattern/format from Responses-format tools for xAI compatibility.
	tools := []ToolDescriptor{
		{
			Name:        "mcp_search",
			Description: "search",
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string"},
					"ts": {"type": "string", "format": "date-time"},
					"domains": {
						"type": "array",
						"items": {
							"type": "string",
							"pattern": "^[a-z]+\\.com$"
						}
					}
				}
			}`),
		},
	}
	got := codexResponsesTools(tools)
	if len(got) != 1 {
		t.Fatalf("got %d tools, want 1", len(got))
	}
	var schema map[string]any
	if err := json.Unmarshal(got[0].Parameters, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	props, _ := schema["properties"].(map[string]any)
	if ts, ok := props["ts"].(map[string]any); ok {
		if _, hasFormat := ts["format"]; hasFormat {
			t.Error("format keyword was not stripped from schema")
		}
	}
	domains, _ := props["domains"].(map[string]any)
	items, _ := domains["items"].(map[string]any)
	if _, hasPattern := items["pattern"]; hasPattern {
		t.Error("pattern keyword was not stripped from nested items schema")
	}
	if items["type"] != "string" {
		t.Errorf("items.type = %v, want string (type should be preserved)", items["type"])
	}
}

func TestCodexResponsesToolSchema_StripsTopLevelCombinators(t *testing.T) {
	// OpenAI Codex / xAI Responses backends reject top-level allOf/anyOf/oneOf/enum/not.
	// Mirrors Hermes fix: strip Codex-hostile top-level schema combinators (3924cb408).
	tools := []ToolDescriptor{
		{
			Name:        "memory",
			Description: "store",
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"action": {"type": "string", "enum": ["add", "replace"]},
					"content": {"type": "string"},
					"nested": {
						"type": "object",
						"properties": {"mode": {"type": "string"}},
						"allOf": [{"required": ["mode"]}]
					}
				},
				"required": ["action"],
				"allOf": [{"if": {"properties": {"action": {"const": "add"}}}, "then": {"required": ["content"]}}],
				"anyOf": [{"required": ["action"]}],
				"oneOf": [{"required": ["action"]}],
				"not": {"required": ["bogus"]},
				"enum": ["bogus-top-level"]
			}`),
		},
	}
	got := codexResponsesTools(tools)
	if len(got) != 1 {
		t.Fatalf("got %d tools, want 1", len(got))
	}
	var schema map[string]any
	if err := json.Unmarshal(got[0].Parameters, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf", "not", "enum"} {
		if _, ok := schema[key]; ok {
			t.Errorf("top-level %q was not stripped", key)
		}
	}
	// Properties and required must survive stripping.
	if _, ok := schema["required"]; !ok {
		t.Error("required was stripped (should be preserved)")
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		t.Fatal("properties missing after stripping")
	}
	// Nested combinator inside a property schema must be preserved.
	nested, _ := props["nested"].(map[string]any)
	if _, ok := nested["allOf"]; !ok {
		t.Error("nested allOf inside a property was stripped (should be preserved)")
	}
}

func TestGrokSupportsReasoningEffort(t *testing.T) {
	// Allowlisted prefixes
	for _, model := range []string{
		"grok-3-mini", "grok-3-mini-fast", "grok-3-mini-2025",
		"grok-4.20-multi-agent-0309", "grok-4.3", "grok-4.3-turbo",
		"x-ai/grok-3-mini", "openrouter/x-ai/grok-4.3",
	} {
		if !grokSupportsReasoningEffort(model) {
			t.Errorf("grokSupportsReasoningEffort(%q) = false, want true", model)
		}
	}
	// Non-allowlisted (should not send effort)
	for _, model := range []string{
		"grok-3", "grok-4", "grok-4-0709", "grok-4-fast",
		"grok-4-1-fast", "grok-code-fast-1", "gpt-5-codex", "",
	} {
		if grokSupportsReasoningEffort(model) {
			t.Errorf("grokSupportsReasoningEffort(%q) = true, want false", model)
		}
	}
}

func TestBuildCodexResponsesPayload_GrokReasoningEffort(t *testing.T) {
	effort := ReasoningEffortHigh
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model:           "grok-3-mini",
		MaxTokens:       1024,
		ReasoningEffort: &effort,
		Messages:        []Message{{Role: "user", Content: "think deeply"}},
	})
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload() error = %v", err)
	}
	if payload.Reasoning == nil || payload.Reasoning.Effort != "high" {
		t.Fatalf("Reasoning = %+v, want {Effort:high}", payload.Reasoning)
	}
	if len(payload.Include) == 0 || payload.Include[0] != "reasoning.encrypted_content" {
		t.Fatalf("Include = %v, want [reasoning.encrypted_content]", payload.Include)
	}
}

func TestBuildCodexResponsesPayload_NonGrokNoReasoningEffort(t *testing.T) {
	effort := ReasoningEffortHigh
	// grok-4 is NOT on the allowlist and should not get reasoning.effort
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model:           "grok-4",
		MaxTokens:       1024,
		ReasoningEffort: &effort,
		Messages:        []Message{{Role: "user", Content: "think"}},
	})
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload() error = %v", err)
	}
	if payload.Reasoning != nil {
		t.Fatalf("Reasoning = %+v, want nil for grok-4 (rejects reasoningEffort)", payload.Reasoning)
	}
	if len(payload.Include) > 0 {
		t.Fatalf("Include = %v, want nil for grok-4", payload.Include)
	}
}
