package hermes

import (
	"encoding/json"
	"testing"
)

func TestDecodeBedrockStream_TextAndUsage(t *testing.T) {
	got, err := decodeBedrockStreamEvents([]map[string]any{
		{"messageStart": map[string]any{"role": "assistant"}},
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 0,
			"delta":             map[string]any{"text": "Hello"},
		}},
		{"contentBlockStop": map[string]any{"contentBlockIndex": 0}},
		{"messageStop": map[string]any{"stopReason": "end_turn"}},
		{"metadata": map[string]any{"usage": map[string]any{"inputTokens": 5, "outputTokens": 3}}},
	}, nil)
	if err != nil {
		t.Fatalf("decodeBedrockStreamEvents() error = %v", err)
	}

	assertTranscriptEvents(t, []eventSnapshot{
		{Kind: EventToken, Token: "Hello"},
		{Kind: EventDone, FinishReason: "stop", TokensIn: 5, TokensOut: 3},
	}, got)
}

func TestDecodeBedrockStream_ReasoningDelta(t *testing.T) {
	got, err := decodeBedrockStreamEvents([]map[string]any{
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 0,
			"delta": map[string]any{
				"reasoningContent": map[string]any{"text": "Let me think..."},
			},
		}},
		{"messageStop": map[string]any{"stopReason": "end_turn"}},
	}, nil)
	if err != nil {
		t.Fatalf("decodeBedrockStreamEvents() error = %v", err)
	}

	assertTranscriptEvents(t, []eventSnapshot{
		{Kind: EventReasoning, Reasoning: "Let me think..."},
		{Kind: EventDone, FinishReason: "stop"},
	}, got)
	for _, ev := range got {
		if ev.Kind == EventToken && ev.Token == "Let me think..." {
			t.Fatalf("reasoning text leaked as token event: %+v", got)
		}
	}
}

func TestDecodeBedrockStream_ToolUseChunks(t *testing.T) {
	got, err := decodeBedrockStreamEvents([]map[string]any{
		{"contentBlockStart": map[string]any{
			"contentBlockIndex": 1,
			"start": map[string]any{"toolUse": map[string]any{
				"toolUseId": "call_1",
				"name":      "read_file",
			}},
		}},
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 1,
			"delta": map[string]any{"toolUse": map[string]any{
				"input": `{"path":`,
			}},
		}},
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 1,
			"delta": map[string]any{"toolUse": map[string]any{
				"input": `"/tmp/f"}`,
			}},
		}},
		{"contentBlockStop": map[string]any{"contentBlockIndex": 1}},
		{"messageStop": map[string]any{"stopReason": "tool_use"}},
	}, nil)
	if err != nil {
		t.Fatalf("decodeBedrockStreamEvents() error = %v", err)
	}

	assertTranscriptEvents(t, []eventSnapshot{{
		Kind:         EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []ToolCall{{
			ID:        "call_1",
			Name:      "read_file",
			Arguments: json.RawMessage(`{"path":"/tmp/f"}`),
		}},
	}}, got)
}

func TestDecodeBedrockStream_MixedTextAndToolPreservesText(t *testing.T) {
	got, err := decodeBedrockStreamEvents([]map[string]any{
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 0,
			"delta":             map[string]any{"text": "I will inspect it."},
		}},
		{"contentBlockStart": map[string]any{
			"contentBlockIndex": 1,
			"start": map[string]any{"toolUse": map[string]any{
				"toolUseId": "call_1",
				"name":      "read_file",
			}},
		}},
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 1,
			"delta": map[string]any{"toolUse": map[string]any{
				"input": `{"path":"/tmp/f"}`,
			}},
		}},
		{"contentBlockStop": map[string]any{"contentBlockIndex": 1}},
		{"messageStop": map[string]any{"stopReason": "tool_use"}},
	}, nil)
	if err != nil {
		t.Fatalf("decodeBedrockStreamEvents() error = %v", err)
	}

	assertTranscriptEvents(t, []eventSnapshot{
		{Kind: EventToken, Token: "I will inspect it."},
		{
			Kind:         EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []ToolCall{{
				ID:        "call_1",
				Name:      "read_file",
				Arguments: json.RawMessage(`{"path":"/tmp/f"}`),
			}},
		},
	}, got)
}

func TestDecodeBedrockStream_EmptyStreamReturnsStop(t *testing.T) {
	got, err := decodeBedrockStreamEvents(nil, nil)
	if err != nil {
		t.Fatalf("decodeBedrockStreamEvents() error = %v", err)
	}

	assertTranscriptEvents(t, []eventSnapshot{{Kind: EventDone, FinishReason: "stop"}}, got)
}

func TestDecodeBedrockStream_InterruptStopsBeforeRemainingEvents(t *testing.T) {
	got, err := decodeBedrockStreamEvents([]map[string]any{
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 0,
			"delta":             map[string]any{"text": "first"},
		}},
		{"contentBlockDelta": map[string]any{
			"contentBlockIndex": 0,
			"delta":             map[string]any{"text": "second"},
		}},
		{"messageStop": map[string]any{"stopReason": "end_turn"}},
	}, func(ev Event) bool {
		return ev.Kind == EventToken && ev.Token == "first"
	})
	if err != nil {
		t.Fatalf("decodeBedrockStreamEvents() error = %v", err)
	}

	assertTranscriptEvents(t, []eventSnapshot{{Kind: EventToken, Token: "first"}}, got)
}
