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

type codexTestRequest struct {
	Model        string                    `json:"model"`
	Instructions string                    `json:"instructions,omitempty"`
	Input        []map[string]any          `json:"input"`
	Stream       bool                      `json:"stream"`
	Tools        []codexTestToolDescriptor `json:"tools,omitempty"`
}

type codexTestToolDescriptor struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

const codexEndTurnFixture = `event: response.output_text.delta
data: {"type":"response.output_text.delta","delta":"done"}

event: response.completed
data: {"type":"response.completed","response":{"usage":{"input_tokens":21,"output_tokens":7},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}]}}

`

const codexToolUseFixture = `event: response.reasoning_summary_text.delta
data: {"type":"response.reasoning_summary_text.delta","delta":"need calculator"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","delta":"Let me check. "}

event: response.output_item.added
data: {"type":"response.output_item.added","item":{"id":"fc_1","type":"function_call","call_id":"call_codex_1","name":"calc","arguments":""}}

event: response.function_call_arguments.delta
data: {"type":"response.function_call_arguments.delta","call_id":"call_codex_1","delta":"{\"expression\":\"2+2\"}"} 

event: response.output_item.done
data: {"type":"response.output_item.done","item":{"id":"fc_1","type":"function_call","call_id":"call_codex_1","name":"calc","arguments":"{\"expression\":\"2+2\"}"}} 

event: response.completed
data: {"type":"response.completed","response":{"usage":{"input_tokens":42,"output_tokens":17},"output":[{"type":"function_call","call_id":"call_codex_1","name":"calc","arguments":"{\"expression\":\"2+2\"}"}]}}

`

func TestNewClient_CodexTranslatesCanonicalMessages(t *testing.T) {
	reqSeen := make(chan codexTestRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %s, want /v1/responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}

		var body codexTestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		reqSeen <- body

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, codexEndTurnFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewClient("codex", srv.URL, "test-key")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model: "gpt-5.3-codex",
		Messages: []Message{
			{Role: "system", Content: "follow rules"},
			{Role: "system", Content: "use tools precisely"},
			{Role: "user", Content: "what is 2+2"},
			{
				Role:    "assistant",
				Content: "I'll calculate that.",
				ToolCalls: []ToolCall{{
					ID:        "call_codex_1",
					Name:      "calc",
					Arguments: json.RawMessage(`{"expression":"2+2"}`),
				}},
			},
			{Role: "tool", ToolCallID: "call_codex_1", Name: "calc", Content: "4"},
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
	if body.Model != "gpt-5.3-codex" {
		t.Fatalf("model = %q, want gpt-5.3-codex", body.Model)
	}
	if !body.Stream {
		t.Fatal("stream = false, want true")
	}
	if body.Instructions != "follow rules\n\nuse tools precisely" {
		t.Fatalf("instructions = %q", body.Instructions)
	}
	if len(body.Input) != 4 {
		t.Fatalf("input len = %d, want 4", len(body.Input))
	}
	if role := body.Input[0]["role"]; role != "user" {
		t.Fatalf("input[0].role = %#v, want user", role)
	}
	if content := body.Input[0]["content"]; content != "what is 2+2" {
		t.Fatalf("input[0].content = %#v, want what is 2+2", content)
	}
	if role := body.Input[1]["role"]; role != "assistant" {
		t.Fatalf("input[1].role = %#v, want assistant", role)
	}
	if content := body.Input[1]["content"]; content != "I'll calculate that." {
		t.Fatalf("input[1].content = %#v, want assistant text", content)
	}
	if typ := body.Input[2]["type"]; typ != "function_call" {
		t.Fatalf("input[2].type = %#v, want function_call", typ)
	}
	if callID := body.Input[2]["call_id"]; callID != "call_codex_1" {
		t.Fatalf("input[2].call_id = %#v, want call_codex_1", callID)
	}
	if name := body.Input[2]["name"]; name != "calc" {
		t.Fatalf("input[2].name = %#v, want calc", name)
	}
	if args := body.Input[2]["arguments"]; args != `{"expression":"2+2"}` {
		t.Fatalf("input[2].arguments = %#v, want JSON args", args)
	}
	if typ := body.Input[3]["type"]; typ != "function_call_output" {
		t.Fatalf("input[3].type = %#v, want function_call_output", typ)
	}
	if callID := body.Input[3]["call_id"]; callID != "call_codex_1" {
		t.Fatalf("input[3].call_id = %#v, want call_codex_1", callID)
	}
	if output := body.Input[3]["output"]; output != "4" {
		t.Fatalf("input[3].output = %#v, want 4", output)
	}
	if len(body.Tools) != 1 || body.Tools[0].Type != "function" || body.Tools[0].Name != "calc" || body.Tools[0].Description != "calculator" {
		t.Fatalf("tools = %+v", body.Tools)
	}
	if got := body.Tools[0].Parameters["type"]; got != "object" {
		t.Fatalf("tool parameters type = %#v, want object", got)
	}
}

func TestNewClient_CodexMapsReasoningAndToolUseEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, codexToolUseFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewClient("openai-codex", srv.URL, "test-key")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "gpt-5.3-codex",
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
		t.Fatalf("finish_reason = %q, want tool_calls", final.FinishReason)
	}
	if final.TokensIn != 42 || final.TokensOut != 17 {
		t.Fatalf("usage = %d/%d, want 42/17", final.TokensIn, final.TokensOut)
	}
	if len(final.ToolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(final.ToolCalls))
	}
	tc := final.ToolCalls[0]
	if tc.ID != "call_codex_1" || tc.Name != "calc" {
		t.Fatalf("tool_call = %+v, want call_codex_1/calc", tc)
	}
	if string(tc.Arguments) != `{"expression":"2+2"}` {
		t.Fatalf("tool_call arguments = %s, want JSON args", tc.Arguments)
	}
}
