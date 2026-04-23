package hermes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type googleCodeAssistTestRequest struct {
	Project      string            `json:"project"`
	Model        string            `json:"model"`
	UserPromptID string            `json:"user_prompt_id"`
	Request      geminiTestRequest `json:"request"`
}

type googleCodeAssistHealthTestRequest struct {
	Metadata struct {
		DuetProject string `json:"duetProject,omitempty"`
		IDEType     string `json:"ideType"`
		Platform    string `json:"platform"`
		PluginType  string `json:"pluginType"`
	} `json:"metadata"`
	CloudAICompanionProject string `json:"cloudaicompanionProject,omitempty"`
}

const googleCodeAssistToolUseFixture = `data: {"response":{"candidates":[{"content":{"role":"model","parts":[{"text":"Need a calculator. "},{"functionCall":{"name":"calc","args":{"expression":"2+2"}}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":42,"candidatesTokenCount":17}}}

data: [DONE]

`

func TestNewClient_GoogleCodeAssistTranslatesCanonicalMessages(t *testing.T) {
	t.Setenv("HERMES_GEMINI_PROJECT_ID", "proj-123")

	reqSeen := make(chan googleCodeAssistTestRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1internal:streamGenerateContent" {
			t.Fatalf("path = %s, want /v1internal:streamGenerateContent", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("alt = %q, want sse", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}

		var body googleCodeAssistTestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		reqSeen <- body

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, geminiEndTurnFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewClient("google-gemini-cli", srv.URL, "test-token")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: "system", Content: "follow rules"},
			{Role: "system", Content: "use tools precisely"},
			{Role: "user", Content: "what is 2+2"},
			{
				Role:    "assistant",
				Content: "I'll calculate that.",
				ToolCalls: []ToolCall{{
					ID:        "call_google_1",
					Name:      "calc",
					Arguments: json.RawMessage(`{"expression":"2+2"}`),
				}},
			},
			{Role: "tool", ToolCallID: "call_google_1", Name: "calc", Content: "4"},
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
	if body.Project != "proj-123" {
		t.Fatalf("project = %q, want proj-123", body.Project)
	}
	if body.Model != "gemini-2.5-flash" {
		t.Fatalf("model = %q, want gemini-2.5-flash", body.Model)
	}
	if body.UserPromptID == "" {
		t.Fatal("user_prompt_id = empty, want generated prompt id")
	}
	if body.Request.SystemInstruction == nil {
		t.Fatal("system_instruction = nil, want combined system prompt")
	}
	if len(body.Request.SystemInstruction.Parts) != 2 {
		t.Fatalf("system_instruction.parts len = %d, want 2", len(body.Request.SystemInstruction.Parts))
	}
	if got := body.Request.SystemInstruction.Parts[0]["text"]; got != "follow rules" {
		t.Fatalf("system_instruction.parts[0].text = %#v, want follow rules", got)
	}
	if got := body.Request.SystemInstruction.Parts[1]["text"]; got != "use tools precisely" {
		t.Fatalf("system_instruction.parts[1].text = %#v, want use tools precisely", got)
	}
	if len(body.Request.Contents) != 3 {
		t.Fatalf("contents len = %d, want 3", len(body.Request.Contents))
	}
	if got := body.Request.Contents[0].Role; got != "user" {
		t.Fatalf("contents[0].role = %q, want user", got)
	}
	if got := body.Request.Contents[0].Parts[0]["text"]; got != "what is 2+2" {
		t.Fatalf("contents[0].parts[0].text = %#v, want what is 2+2", got)
	}
	if got := body.Request.Contents[1].Role; got != "model" {
		t.Fatalf("contents[1].role = %q, want model", got)
	}
	if got := body.Request.Contents[1].Parts[0]["text"]; got != "I'll calculate that." {
		t.Fatalf("contents[1].parts[0].text = %#v, want assistant text", got)
	}
	call, ok := body.Request.Contents[1].Parts[1]["functionCall"].(map[string]any)
	if !ok {
		t.Fatalf("contents[1].parts[1] = %#v, want functionCall", body.Request.Contents[1].Parts[1])
	}
	if got := call["id"]; got != "call_google_1" {
		t.Fatalf("functionCall.id = %#v, want call_google_1", got)
	}
	if got := call["name"]; got != "calc" {
		t.Fatalf("functionCall.name = %#v, want calc", got)
	}
	response, ok := body.Request.Contents[2].Parts[0]["functionResponse"].(map[string]any)
	if !ok {
		t.Fatalf("contents[2].parts[0] = %#v, want functionResponse", body.Request.Contents[2].Parts[0])
	}
	if got := response["id"]; got != "call_google_1" {
		t.Fatalf("functionResponse.id = %#v, want call_google_1", got)
	}
	if got := response["name"]; got != "calc" {
		t.Fatalf("functionResponse.name = %#v, want calc", got)
	}
	if len(body.Request.Tools) != 1 || len(body.Request.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("tools = %+v, want one function declaration", body.Request.Tools)
	}
}

func TestNewClient_GoogleCodeAssistMapsWrappedToolCallEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, googleCodeAssistToolUseFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewClient("google-code-assist", srv.URL, "test-token")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "gemini-2.5-flash",
		Messages: []Message{{Role: "user", Content: "what is 2+2?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var got []Event
	for {
		ev, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, ev)
		if ev.Kind == EventDone {
			break
		}
	}

	if len(got) != 2 {
		t.Fatalf("event count = %d, want 2", len(got))
	}
	if got[0].Kind != EventToken || got[0].Token != "Need a calculator. " {
		t.Fatalf("got[0] = %+v, want text token", got[0])
	}
	final := got[1]
	if final.Kind != EventDone {
		t.Fatalf("got[1] = %+v, want done event", final)
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
	if tc.Name != "calc" {
		t.Fatalf("tool_call = %+v, want calc", tc)
	}
	if string(tc.Arguments) != `{"expression":"2+2"}` {
		t.Fatalf("tool_call arguments = %s, want JSON args", tc.Arguments)
	}
}

func TestNewClient_GoogleCodeAssistHealthUsesLoadCodeAssist(t *testing.T) {
	t.Setenv("HERMES_GEMINI_PROJECT_ID", "proj-123")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1internal:loadCodeAssist" {
			t.Fatalf("path = %s, want /v1internal:loadCodeAssist", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}

		var body googleCodeAssistHealthTestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Metadata.DuetProject != "proj-123" {
			t.Fatalf("metadata.duetProject = %q, want proj-123", body.Metadata.DuetProject)
		}
		if body.CloudAICompanionProject != "proj-123" {
			t.Fatalf("cloudaicompanionProject = %q, want proj-123", body.CloudAICompanionProject)
		}
		if body.Metadata.PluginType != "GEMINI" {
			t.Fatalf("metadata.pluginType = %q, want GEMINI", body.Metadata.PluginType)
		}

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"currentTier":{"id":"free-tier"}}`))
	}))
	defer srv.Close()

	c := NewClient("google-gemini-cli", srv.URL, "test-token")
	if err := c.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
}
