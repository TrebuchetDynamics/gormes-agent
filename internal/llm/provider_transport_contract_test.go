package llm

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestProviderTransportRegistryAndFixtureHarness(t *testing.T) {
	lookupReq := ChatRequest{
		Model:     "fixture-model",
		MaxTokens: 128,
		Stream:    true,
		Messages: []Message{
			{Role: "system", Content: "Follow policy."},
			{Role: "user", Content: "lookup status"},
		},
		Tools: []ToolDescriptor{{
			Name:        "lookup",
			Description: "Looks up fixture status.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		}},
	}
	echoReq := lookupReq
	echoReq.Tools = []ToolDescriptor{{
		Name:        "echo",
		Description: "Echoes text.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	}}
	weatherReq := lookupReq
	weatherReq.Tools = []ToolDescriptor{{
		Name:        "get_weather",
		Description: "Returns weather.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
	}}

	tests := []struct {
		name       string
		apiMode    string
		wantPath   string
		request    ChatRequest
		fixture    string
		wantEvents []eventSnapshot
	}{
		{
			name:     "chat completions",
			apiMode:  "chat_completions",
			wantPath: defaultChatCompletionsPath,
			request:  echoReq,
			fixture:  readProviderFixture(t, "openai_tool_call_eof.sse"),
			wantEvents: []eventSnapshot{{
				Kind:         EventDone,
				FinishReason: "tool_calls",
				ToolCalls: []ToolCall{{
					ID:        "call_echo",
					Name:      "echo",
					Arguments: json.RawMessage(`{"text":"hi"}`),
				}},
			}},
		},
		{
			name:     "anthropic messages",
			apiMode:  "anthropic_messages",
			wantPath: defaultAnthropicMessagesPath,
			request:  weatherReq,
			fixture:  readProviderFixture(t, "anthropic_tool_use.sse"),
			wantEvents: []eventSnapshot{
				{Kind: EventReasoning, Reasoning: "Need a tool."},
				{Kind: EventToken, Token: "Checking weather. "},
				{Kind: EventToken, Token: "One moment."},
				{
					Kind:         EventDone,
					FinishReason: "tool_calls",
					TokensIn:     11,
					TokensOut:    23,
					ToolCalls: []ToolCall{{
						ID:        "toolu_1",
						Name:      "get_weather",
						Arguments: json.RawMessage(`{"location":"Monterrey"}`),
					}},
				},
			},
		},
		{
			name:     "bedrock converse",
			apiMode:  "bedrock_converse",
			wantPath: "bedrock:ConverseStream",
			request:  lookupReq,
			fixture: `[
				{"contentBlockDelta":{"contentBlockIndex":0,"delta":{"reasoningContent":{"text":"Need lookup."}}}},
				{"contentBlockDelta":{"contentBlockIndex":0,"delta":{"text":"Checking. "}}},
				{"contentBlockStart":{"contentBlockIndex":1,"start":{"toolUse":{"toolUseId":"toolu_lookup","name":"lookup"}}}},
				{"contentBlockDelta":{"contentBlockIndex":1,"delta":{"toolUse":{"input":"{\"query\":\"status\"}"}}}},
				{"messageStop":{"stopReason":"tool_use"}},
				{"metadata":{"usage":{"inputTokens":13,"outputTokens":7}}}
			]`,
			wantEvents: []eventSnapshot{
				{Kind: EventReasoning, Reasoning: "Need lookup."},
				{Kind: EventToken, Token: "Checking. "},
				{
					Kind:         EventDone,
					FinishReason: "tool_calls",
					TokensIn:     13,
					TokensOut:    7,
					ToolCalls: []ToolCall{{
						ID:        "toolu_lookup",
						Name:      "lookup",
						Arguments: json.RawMessage(`{"query":"status"}`),
					}},
				},
			},
		},
		{
			name:     "codex responses",
			apiMode:  "codex_responses",
			wantPath: "/v1/responses",
			request:  lookupReq,
			fixture: `{
				"status":"completed",
				"output":[
					{"type":"reasoning","summary":[{"type":"summary_text","text":"Need status lookup."}]},
					{"type":"message","status":"completed","content":[{"type":"output_text","text":"Checking status."}]},
					{"type":"function_call","id":"fc_lookup","call_id":"call_lookup","name":"lookup","arguments":{"query":"status"}}
				],
				"usage":{"input_tokens":21,"output_tokens":8,"total_tokens":29}
			}`,
			wantEvents: []eventSnapshot{
				{Kind: EventReasoning, Reasoning: "Need status lookup."},
				{Kind: EventToken, Token: "Checking status."},
				{
					Kind:         EventDone,
					FinishReason: "tool_calls",
					TokensIn:     21,
					TokensOut:    8,
					ToolCalls: []ToolCall{{
						ID:        "call_lookup",
						Name:      "lookup",
						Arguments: json.RawMessage(`{"query":"status"}`),
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, ok := GetProviderTransport(tt.apiMode)
			if !ok {
				t.Fatalf("GetProviderTransport(%q) not found", tt.apiMode)
			}
			if transport.APIMode() != tt.apiMode {
				t.Fatalf("APIMode() = %q, want %q", transport.APIMode(), tt.apiMode)
			}

			request, err := transport.BuildRequest(tt.request)
			if err != nil {
				t.Fatalf("BuildRequest() error = %v", err)
			}
			if request.APIMode != tt.apiMode {
				t.Fatalf("request APIMode = %q, want %q", request.APIMode, tt.apiMode)
			}
			if request.Path != tt.wantPath {
				t.Fatalf("request path = %q, want %q", request.Path, tt.wantPath)
			}
			if len(request.Body) == 0 {
				t.Fatal("request body is empty")
			}

			stream, err := transport.OpenFixtureStream(io.NopCloser(strings.NewReader(tt.fixture)), request)
			if err != nil {
				t.Fatalf("OpenFixtureStream() error = %v", err)
			}
			defer stream.Close()
			assertTranscriptEvents(t, tt.wantEvents, collectStreamEvents(t, stream))
		})
	}
}

func TestProviderTransportRegistryUnknownMode(t *testing.T) {
	if _, ok := GetProviderTransport("unknown_mode"); ok {
		t.Fatal("GetProviderTransport(unknown_mode) found transport, want missing")
	}
}

func TestProviderTransportStaticStreamHonorsContextCancellation(t *testing.T) {
	stream := newStaticProviderStream([]Event{{Kind: EventToken, Token: "hello"}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stream.Recv(ctx); err != context.Canceled {
		t.Fatalf("Recv canceled err = %v, want context.Canceled", err)
	}
}
