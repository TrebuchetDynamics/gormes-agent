package hermes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type providerTranscript struct {
	name          string
	streamFixture string
	request       ChatRequest
	newClient     func(baseURL string) Client
	assertRequest func(t *testing.T, got capturedProviderRequest)
	wantEvents    []eventSnapshot
}

type capturedProviderRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   []byte
}

type eventSnapshot struct {
	Kind         EventKind
	Token        string
	Reasoning    string
	FinishReason string
	TokensIn     int
	TokensOut    int
	ToolCalls    []ToolCall
}

func TestProviderTranscriptHarness_OpenAICompatibleFlushesPendingToolCallOnEOF(t *testing.T) {
	runProviderTranscript(t, providerTranscript{
		name:          "openai-compatible pending tool call EOF",
		streamFixture: "openai_tool_call_eof.sse",
		request: ChatRequest{
			Model:  "fixture-model",
			Stream: true,
			Messages: []Message{
				{Role: "user", Content: "echo hi"},
			},
			Tools: []ToolDescriptor{{
				Name:        "echo",
				Description: "Echoes text",
				Schema:      json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
			}},
		},
		newClient: func(baseURL string) Client {
			return NewHTTPClient(baseURL, "")
		},
		assertRequest: func(t *testing.T, got capturedProviderRequest) {
			t.Helper()
			if got.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", got.Method)
			}
			if got.Path != "/v1/chat/completions" {
				t.Fatalf("path = %q, want /v1/chat/completions", got.Path)
			}
			var body struct {
				Model    string `json:"model"`
				Stream   bool   `json:"stream"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
				Tools []struct {
					Type     string `json:"type"`
					Function struct {
						Name string `json:"name"`
					} `json:"function"`
				} `json:"tools"`
			}
			if err := json.Unmarshal(got.Body, &body); err != nil {
				t.Fatalf("decode request body: %v\n%s", err, got.Body)
			}
			if body.Model != "fixture-model" || !body.Stream {
				t.Fatalf("request model/stream = %q/%v, want fixture-model/true", body.Model, body.Stream)
			}
			if len(body.Messages) != 1 || body.Messages[0].Role != "user" || body.Messages[0].Content != "echo hi" {
				t.Fatalf("messages = %+v, want single user echo request", body.Messages)
			}
			if len(body.Tools) != 1 || body.Tools[0].Type != "function" || body.Tools[0].Function.Name != "echo" {
				t.Fatalf("tools = %+v, want echo function descriptor", body.Tools)
			}
		},
		wantEvents: []eventSnapshot{{
			Kind:         EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []ToolCall{{
				ID:        "call_echo",
				Name:      "echo",
				Arguments: json.RawMessage(`{"text":"hi"}`),
			}},
		}},
	})
}

func TestProviderTranscriptHarness_OpenAICompatibleMapsToolContinuationRequest(t *testing.T) {
	runProviderTranscript(t, providerTranscript{
		name:          "openai-compatible tool continuation request",
		streamFixture: "openai_stop.sse",
		request: ChatRequest{
			Model:  "fixture-model",
			Stream: true,
			Messages: []Message{
				{Role: "user", Content: "echo hi"},
				{
					Role:    "assistant",
					Content: "Calling echo.",
					ToolCalls: []ToolCall{{
						ID:        "call_echo",
						Name:      "echo",
						Arguments: json.RawMessage(`{"text":"hi"}`),
					}},
				},
				{Role: "tool", ToolCallID: "call_echo", Name: "echo", Content: "hi"},
			},
			Tools: []ToolDescriptor{{
				Name:        "echo",
				Description: "Echoes text",
				Schema:      json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
			}},
		},
		newClient: func(baseURL string) Client {
			return NewHTTPClient(baseURL, "")
		},
		assertRequest: func(t *testing.T, got capturedProviderRequest) {
			t.Helper()
			if got.Path != "/v1/chat/completions" {
				t.Fatalf("path = %q, want /v1/chat/completions", got.Path)
			}
			var body struct {
				Model    string `json:"model"`
				Stream   bool   `json:"stream"`
				Messages []struct {
					Role       string `json:"role"`
					Content    string `json:"content,omitempty"`
					ToolCallID string `json:"tool_call_id,omitempty"`
					Name       string `json:"name,omitempty"`
					ToolCalls  []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(got.Body, &body); err != nil {
				t.Fatalf("decode request body: %v\n%s", err, got.Body)
			}
			if body.Model != "fixture-model" || !body.Stream {
				t.Fatalf("request model/stream = %q/%v, want fixture-model/true", body.Model, body.Stream)
			}
			if len(body.Messages) != 3 {
				t.Fatalf("messages len = %d, want 3: %s", len(body.Messages), got.Body)
			}
			assistant := body.Messages[1]
			if assistant.Role != "assistant" || assistant.Content != "Calling echo." {
				t.Fatalf("assistant message = %+v, want role/content preserved", assistant)
			}
			if len(assistant.ToolCalls) != 1 {
				t.Fatalf("assistant tool_calls len = %d, want 1: %+v", len(assistant.ToolCalls), assistant.ToolCalls)
			}
			call := assistant.ToolCalls[0]
			if call.ID != "call_echo" || call.Type != "function" || call.Function.Name != "echo" || call.Function.Arguments != `{"text":"hi"}` {
				t.Fatalf("assistant tool call = %+v, want call_echo echo payload", call)
			}
			toolReply := body.Messages[2]
			if toolReply.Role != "tool" || toolReply.ToolCallID != "call_echo" || toolReply.Content != "hi" {
				t.Fatalf("tool reply = %+v, want linked tool result payload", toolReply)
			}
		},
		wantEvents: []eventSnapshot{{
			Kind:         EventDone,
			FinishReason: "stop",
			TokensIn:     21,
			TokensOut:    3,
		}},
	})
}

func TestProviderTranscriptHarness_AnthropicFlushesPendingToolCallOnEOF(t *testing.T) {
	runProviderTranscript(t, providerTranscript{
		name:          "anthropic pending tool call EOF",
		streamFixture: "anthropic_tool_call_eof.sse",
		request: ChatRequest{
			Model:  "claude-fixture",
			Stream: true,
			Messages: []Message{
				{Role: "user", Content: "weather in Monterrey"},
			},
			Tools: []ToolDescriptor{{
				Name:        "get_weather",
				Description: "Returns the weather",
				Schema:      json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
			}},
		},
		newClient: func(baseURL string) Client {
			return NewAnthropicClient(baseURL, "sk-ant-api-test")
		},
		assertRequest: func(t *testing.T, got capturedProviderRequest) {
			t.Helper()
			if got.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", got.Method)
			}
			if got.Path != "/v1/messages" {
				t.Fatalf("path = %q, want /v1/messages", got.Path)
			}
			if got.Header.Get("x-api-key") != "sk-ant-api-test" {
				t.Fatalf("x-api-key = %q, want sk-ant-api-test", got.Header.Get("x-api-key"))
			}
			if got.Header.Get("anthropic-version") != anthropicVersion {
				t.Fatalf("anthropic-version = %q, want %s", got.Header.Get("anthropic-version"), anthropicVersion)
			}
			var body struct {
				Model    string `json:"model"`
				Stream   bool   `json:"stream"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			}
			if err := json.Unmarshal(got.Body, &body); err != nil {
				t.Fatalf("decode request body: %v\n%s", err, got.Body)
			}
			if body.Model != "claude-fixture" || !body.Stream {
				t.Fatalf("request model/stream = %q/%v, want claude-fixture/true", body.Model, body.Stream)
			}
			if len(body.Messages) != 1 || body.Messages[0].Role != "user" || body.Messages[0].Content != "weather in Monterrey" {
				t.Fatalf("messages = %+v, want single user weather request", body.Messages)
			}
			if len(body.Tools) != 1 || body.Tools[0].Name != "get_weather" {
				t.Fatalf("tools = %+v, want get_weather descriptor", body.Tools)
			}
		},
		wantEvents: []eventSnapshot{{
			Kind:         EventDone,
			FinishReason: "tool_calls",
			TokensIn:     17,
			ToolCalls: []ToolCall{{
				ID:        "toolu_partial",
				Name:      "get_weather",
				Arguments: json.RawMessage(`{"location":"Monterrey"}`),
			}},
		}},
	})
}

func TestProviderTranscriptHarness_AnthropicReplaysRequestStreamFinishAndUsage(t *testing.T) {
	runProviderTranscript(t, providerTranscript{
		name:          "anthropic request and stream transcript",
		streamFixture: "anthropic_tool_use.sse",
		request: ChatRequest{
			Model:     "claude-fixture",
			MaxTokens: 512,
			Stream:    true,
			Messages: []Message{
				{Role: "system", Content: "cached system", CacheControl: &CacheControl{Type: "ephemeral"}},
				{Role: "user", Content: "weather in Monterrey"},
				{
					Role:    "assistant",
					Content: "Checking.",
					ToolCalls: []ToolCall{{
						ID:        "toolu_prior",
						Name:      "get_weather",
						Arguments: json.RawMessage(`{"location":"Monterrey"}`),
					}},
				},
				{Role: "tool", ToolCallID: "toolu_prior", Name: "get_weather", Content: "72F", CacheControl: &CacheControl{Type: "ephemeral"}},
			},
			Tools: []ToolDescriptor{{
				Name:        "get_weather",
				Description: "Returns the weather",
				Schema:      json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
			}},
		},
		newClient: func(baseURL string) Client {
			return NewAnthropicClient(baseURL, "sk-ant-api-test")
		},
		assertRequest: func(t *testing.T, got capturedProviderRequest) {
			t.Helper()
			if got.Path != "/v1/messages" {
				t.Fatalf("path = %q, want /v1/messages", got.Path)
			}
			var body struct {
				Model     string `json:"model"`
				MaxTokens int    `json:"max_tokens"`
				Stream    bool   `json:"stream"`
				System    []struct {
					Type         string `json:"type"`
					Text         string `json:"text"`
					CacheControl struct {
						Type string `json:"type"`
					} `json:"cache_control"`
				} `json:"system"`
				Messages []struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"messages"`
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			}
			if err := json.Unmarshal(got.Body, &body); err != nil {
				t.Fatalf("decode request body: %v\n%s", err, got.Body)
			}
			if body.Model != "claude-fixture" || body.MaxTokens != 512 || !body.Stream {
				t.Fatalf("request model/max_tokens/stream = %q/%d/%v", body.Model, body.MaxTokens, body.Stream)
			}
			if len(body.System) != 1 || body.System[0].Text != "cached system" || body.System[0].CacheControl.Type != "ephemeral" {
				t.Fatalf("system blocks = %+v, want cached system with cache_control", body.System)
			}
			if len(body.Messages) != 3 {
				t.Fatalf("messages len = %d, want 3 after tool-result continuation mapping", len(body.Messages))
			}
			var userContent string
			if err := json.Unmarshal(body.Messages[0].Content, &userContent); err != nil || userContent != "weather in Monterrey" {
				t.Fatalf("user content = %s (err=%v), want weather prompt", body.Messages[0].Content, err)
			}
			var assistantContent []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
				ID   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			}
			if err := json.Unmarshal(body.Messages[1].Content, &assistantContent); err != nil {
				t.Fatalf("decode assistant content: %v\n%s", err, body.Messages[1].Content)
			}
			if len(assistantContent) != 2 || assistantContent[1].Type != "tool_use" || assistantContent[1].Name != "get_weather" {
				t.Fatalf("assistant content = %+v, want text + get_weather tool_use", assistantContent)
			}
			var toolResultContent []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id,omitempty"`
			}
			if err := json.Unmarshal(body.Messages[2].Content, &toolResultContent); err != nil {
				t.Fatalf("decode tool-result content: %v\n%s", err, body.Messages[2].Content)
			}
			if len(toolResultContent) != 1 || toolResultContent[0].Type != "tool_result" || toolResultContent[0].ToolUseID != "toolu_prior" {
				t.Fatalf("tool-result content = %+v, want linked tool_result", toolResultContent)
			}
			if len(body.Tools) != 1 || body.Tools[0].Name != "get_weather" {
				t.Fatalf("tools = %+v, want get_weather descriptor", body.Tools)
			}
		},
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
	})
}

func runProviderTranscript(t *testing.T, transcript providerTranscript) {
	t.Helper()
	body := readProviderFixture(t, transcript.streamFixture)
	var captured capturedProviderRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("%s: read request body: %v", transcript.name, err)
		}
		captured = capturedProviderRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Header: r.Header.Clone(),
			Body:   raw,
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, body)
		if err := bw.Flush(); err != nil {
			t.Fatalf("%s: flush fixture: %v", transcript.name, err)
		}
	}))
	defer srv.Close()

	client := transcript.newClient(srv.URL)
	stream, err := client.OpenStream(context.Background(), transcript.request)
	if err != nil {
		t.Fatalf("%s: OpenStream() error = %v", transcript.name, err)
	}
	defer stream.Close()

	if transcript.assertRequest != nil {
		transcript.assertRequest(t, captured)
	}
	assertTranscriptEvents(t, transcript.wantEvents, collectStreamEvents(t, stream))
}

func readProviderFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "provider_transcripts", name))
	if err != nil {
		t.Fatalf("read provider fixture %s: %v", name, err)
	}
	return string(raw)
}

func collectStreamEvents(t *testing.T, stream Stream) []Event {
	t.Helper()
	var events []Event
	for {
		ev, err := stream.Recv(context.Background())
		if err == io.EOF {
			return events
		}
		if err != nil {
			t.Fatalf("Recv() error = %v", err)
		}
		events = append(events, ev)
	}
}

func assertTranscriptEvents(t *testing.T, want []eventSnapshot, got []Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("event count = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Kind != want[i].Kind ||
			got[i].Token != want[i].Token ||
			got[i].Reasoning != want[i].Reasoning ||
			got[i].FinishReason != want[i].FinishReason ||
			got[i].TokensIn != want[i].TokensIn ||
			got[i].TokensOut != want[i].TokensOut {
			t.Fatalf("event[%d] = %+v, want %+v", i, got[i], want[i])
		}
		assertToolCalls(t, i, want[i].ToolCalls, got[i].ToolCalls)
	}
}

func assertToolCalls(t *testing.T, eventIndex int, want, got []ToolCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("event[%d].ToolCalls len = %d, want %d: %+v", eventIndex, len(got), len(want), got)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Name != want[i].Name {
			t.Fatalf("event[%d].ToolCalls[%d] = %+v, want id/name %+v", eventIndex, i, got[i], want[i])
		}
		var gotArgs, wantArgs any
		if err := json.Unmarshal(got[i].Arguments, &gotArgs); err != nil {
			t.Fatalf("event[%d].ToolCalls[%d].Arguments invalid JSON: %v: %s", eventIndex, i, err, got[i].Arguments)
		}
		if err := json.Unmarshal(want[i].Arguments, &wantArgs); err != nil {
			t.Fatalf("test fixture expected arguments invalid JSON: %v: %s", err, want[i].Arguments)
		}
		if !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Fatalf("event[%d].ToolCalls[%d].Arguments = %s, want %s", eventIndex, i, got[i].Arguments, want[i].Arguments)
		}
	}
}
