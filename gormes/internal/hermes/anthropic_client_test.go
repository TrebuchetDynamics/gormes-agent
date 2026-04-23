package hermes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type anthropicTestRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	System    string `json:"system,omitempty"`
	Stream    bool   `json:"stream"`
	Messages  []struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string         `json:"type"`
			Text      string         `json:"text,omitempty"`
			ID        string         `json:"id,omitempty"`
			Name      string         `json:"name,omitempty"`
			Input     map[string]any `json:"input,omitempty"`
			ToolUseID string         `json:"tool_use_id,omitempty"`
			Content   string         `json:"content,omitempty"`
		} `json:"content"`
	} `json:"messages"`
	Tools []struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		InputSchema map[string]any `json:"input_schema"`
	} `json:"tools,omitempty"`
}

const anthropicEndTurnFixture = `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":21,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"done"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":7}}

event: message_stop
data: {"type":"message_stop"}

`

const anthropicToolUseFixture = `event: message_start
data: {"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":42,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"need calculator"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Let me check. "}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: content_block_start
data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"toolu_stream","name":"calc","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"expression\":\"2+2\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":2}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":17}}

event: message_stop
data: {"type":"message_stop"}

`

func TestNewClient_AnthropicTranslatesCanonicalMessages(t *testing.T) {
	reqSeen := make(chan anthropicTestRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %s, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("anthropic-version = %q, want 2023-06-01", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}

		var body anthropicTestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		reqSeen <- body

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, anthropicEndTurnFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewClient("anthropic", srv.URL, "test-key")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []Message{
			{Role: "system", Content: "follow rules"},
			{Role: "system", Content: "use tools precisely"},
			{Role: "user", Content: "what is 2+2"},
			{
				Role:    "assistant",
				Content: "I'll calculate that.",
				ToolCalls: []ToolCall{{
					ID:        "toolu_123",
					Name:      "calc",
					Arguments: json.RawMessage(`{"expression":"2+2"}`),
				}},
			},
			{Role: "tool", ToolCallID: "toolu_123", Name: "calc", Content: "4"},
		},
		Tools: []ToolDescriptor{{
			Name:        "calc",
			Description: "calculator",
			Schema:      json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string"}},"required":["expression"]}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for {
		ev, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if ev.Kind == EventDone {
			break
		}
	}

	body := <-reqSeen
	if body.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("model = %q, want claude-sonnet-4-20250514", body.Model)
	}
	if body.MaxTokens <= 0 {
		t.Fatalf("max_tokens = %d, want > 0", body.MaxTokens)
	}
	if !body.Stream {
		t.Fatal("stream = false, want true")
	}
	if body.System != "follow rules\n\nuse tools precisely" {
		t.Fatalf("system = %q", body.System)
	}
	if len(body.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(body.Messages))
	}
	if body.Messages[0].Role != "user" || len(body.Messages[0].Content) != 1 || body.Messages[0].Content[0].Text != "what is 2+2" {
		t.Fatalf("messages[0] = %+v", body.Messages[0])
	}
	if body.Messages[1].Role != "assistant" || len(body.Messages[1].Content) != 2 {
		t.Fatalf("messages[1] = %+v", body.Messages[1])
	}
	if body.Messages[1].Content[0].Type != "text" || body.Messages[1].Content[0].Text != "I'll calculate that." {
		t.Fatalf("assistant text block = %+v", body.Messages[1].Content[0])
	}
	if body.Messages[1].Content[1].Type != "tool_use" || body.Messages[1].Content[1].ID != "toolu_123" || body.Messages[1].Content[1].Name != "calc" {
		t.Fatalf("assistant tool block = %+v", body.Messages[1].Content[1])
	}
	if got := body.Messages[1].Content[1].Input["expression"]; got != "2+2" {
		t.Fatalf("tool input expression = %#v, want 2+2", got)
	}
	if body.Messages[2].Role != "user" || len(body.Messages[2].Content) != 1 {
		t.Fatalf("messages[2] = %+v", body.Messages[2])
	}
	if body.Messages[2].Content[0].Type != "tool_result" || body.Messages[2].Content[0].ToolUseID != "toolu_123" || body.Messages[2].Content[0].Content != "4" {
		t.Fatalf("tool result block = %+v", body.Messages[2].Content[0])
	}
	if len(body.Tools) != 1 || body.Tools[0].Name != "calc" || body.Tools[0].Description != "calculator" {
		t.Fatalf("tools = %+v", body.Tools)
	}
	if got := body.Tools[0].InputSchema["type"]; got != "object" {
		t.Fatalf("tool input_schema type = %#v, want object", got)
	}
}

func TestNewClient_AnthropicMapsThinkingAndToolUseEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, anthropicToolUseFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewClient("anthropic", srv.URL, "test-key")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: "user", Content: "what is 2+2?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var reasoning strings.Builder
	var tokens strings.Builder
	var final Event
	for {
		ev, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch ev.Kind {
		case EventReasoning:
			reasoning.WriteString(ev.Reasoning)
		case EventToken:
			tokens.WriteString(ev.Token)
		case EventDone:
			final = ev
			goto done
		}
	}

done:
	if reasoning.String() != "need calculator" {
		t.Fatalf("reasoning = %q, want need calculator", reasoning.String())
	}
	if tokens.String() != "Let me check. " {
		t.Fatalf("tokens = %q, want Let me check. ", tokens.String())
	}
	if final.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want tool_calls", final.FinishReason)
	}
	if final.TokensIn != 42 || final.TokensOut != 17 {
		t.Fatalf("usage = %d/%d, want 42/17", final.TokensIn, final.TokensOut)
	}
	if len(final.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(final.ToolCalls))
	}
	if final.ToolCalls[0].ID != "toolu_stream" || final.ToolCalls[0].Name != "calc" {
		t.Fatalf("ToolCalls[0] = %+v", final.ToolCalls[0])
	}
	if !strings.Contains(string(final.ToolCalls[0].Arguments), `"2+2"`) {
		t.Fatalf("ToolCalls[0].Arguments = %s, want expression JSON", final.ToolCalls[0].Arguments)
	}
}

func TestNewClient_AnthropicHealthUsesModelsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %s, want /v1/models", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("anthropic-version = %q, want 2023-06-01", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := NewClient("anthropic", srv.URL, "test-key")
	if err := c.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
}
