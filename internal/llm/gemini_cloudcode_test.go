package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestGeminiCloudCodeBuildRequestMapsMessagesAndSystem(t *testing.T) {
	temp := 0.5
	req := ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi", ToolCalls: []ToolCall{{ID: "call_1", Name: "search", Arguments: json.RawMessage(`{"q":"test"}`)}}},
			{Role: "tool", Content: `{"result":"ok"}`, ToolCallID: "call_1", Name: "search"},
		},
		Temperature: &temp,
		MaxTokens:   100,
		Tools: []ToolDescriptor{
			{Name: "search", Description: "Search the web", Schema: json.RawMessage(`{"type":"object"}`)},
		},
	}
	transport := geminiCloudCodeTransport{}
	pr, err := transport.BuildRequest(req)
	if err != nil {
		t.Fatalf("BuildRequest error = %v", err)
	}
	if pr.APIMode != geminiCloudCodeAPIMode {
		t.Errorf("APIMode = %q, want %q", pr.APIMode, geminiCloudCodeAPIMode)
	}
	var payload geminiRequest
	if err := json.Unmarshal(pr.Body, &payload); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}
	if payload.SystemInstruction == nil || len(payload.SystemInstruction.Parts) != 1 || payload.SystemInstruction.Parts[0].Text != "You are helpful" {
		t.Error("system instruction not mapped correctly")
	}
	if len(payload.Contents) != 3 {
		t.Errorf("contents len = %d, want 3", len(payload.Contents))
	}
	if payload.GenerationConfig.Temperature != 0.5 {
		t.Errorf("temperature = %v, want 0.5", payload.GenerationConfig.Temperature)
	}
	if payload.GenerationConfig.MaxOutputTokens != 100 {
		t.Errorf("maxOutputTokens = %d, want 100", payload.GenerationConfig.MaxOutputTokens)
	}
	if len(payload.Tools) != 1 {
		t.Errorf("tools len = %d, want 1", len(payload.Tools))
	}
}

func TestGeminiCloudCodeStreamEventNormalization(t *testing.T) {
	events := []geminiStreamEvent{
		{Candidates: []geminiCandidate{{Content: geminiContent{Parts: []geminiPart{{Text: "Hello"}}}}}},
		{Candidates: []geminiCandidate{{Content: geminiContent{Parts: []geminiPart{{Text: " world"}}}}}},
		{Candidates: []geminiCandidate{{Content: geminiContent{Parts: []geminiPart{{FunctionCall: &geminiFunctionCall{Name: "search", Args: json.RawMessage(`{"q":"test"}`)}}}}}}},
	}
	raw, _ := json.Marshal(events)
	stream := newGeminiCloudCodeStream(raw, nil)

	ev, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv error = %v", err)
	}
	if ev.Kind != EventToken || ev.Token != "Hello" {
		t.Errorf("first event = %+v, want EventToken Hello", ev)
	}

	ev, _ = stream.Recv(context.Background())
	if ev.Kind != EventToken || ev.Token != " world" {
		t.Errorf("second event = %+v, want EventToken ' world'", ev)
	}

	ev, _ = stream.Recv(context.Background())
	if ev.Kind != EventDone || len(ev.ToolCalls) != 1 || ev.ToolCalls[0].Name != "search" {
		t.Errorf("third event = %+v, want EventDone with tool_calls", ev)
	}
}

func TestGeminiCloudCodeHTTPErrorClassification(t *testing.T) {
	tests := []struct {
		status int
		want   ErrorClass
	}{
		{400, ClassFatal},
		{401, ClassFatal},
		{403, ClassFatal},
		{404, ClassFatal},
		{429, ClassRetryable},
		{500, ClassRetryable},
		{503, ClassRetryable},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			class := classifyGeminiCloudCodeError(tt.status, `{"error":{"message":"test"}}`, http.Header{})
			if class.Class != tt.want {
				t.Errorf("class = %v, want %v", class.Class, tt.want)
			}
		})
	}
}

func TestGeminiCloudCodeProviderStatus(t *testing.T) {
	status := geminiCloudCodeProviderStatus()
	if status.Provider != "gemini_cloudcode" {
		t.Errorf("Provider = %q, want gemini_cloudcode", status.Provider)
	}
	if status.Runtime != geminiCloudCodeAPIMode {
		t.Errorf("Runtime = %q, want %q", status.Runtime, geminiCloudCodeAPIMode)
	}
}
