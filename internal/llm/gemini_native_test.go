package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiNativeTransportRegistered(t *testing.T) {
	transport, ok := GetProviderTransport(geminiNativeAPIMode)
	if !ok {
		t.Fatalf("GetProviderTransport(%q) ok = false", geminiNativeAPIMode)
	}
	if transport.APIMode() != geminiNativeAPIMode {
		t.Fatalf("APIMode = %q, want %q", transport.APIMode(), geminiNativeAPIMode)
	}
}

func TestGeminiNativeBuildRequestMapsMessagesAndTools(t *testing.T) {
	temp := 0.25
	transport := geminiNativeTransport{}
	pr, err := transport.BuildRequest(ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: "system", Content: "You are concise"},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Calling", ToolCalls: []ToolCall{{Name: "search", Arguments: json.RawMessage(`{"q":"abc"}`)}}},
			{Role: "tool", Name: "search", ToolCallID: "call_1", Content: `{"result":"ok"}`},
		},
		Temperature: &temp,
		MaxTokens:   512,
		Tools:       []ToolDescriptor{{Name: "search", Description: "Search", Schema: json.RawMessage(`{"type":"object"}`)}},
	})
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}
	if pr.APIMode != geminiNativeAPIMode {
		t.Fatalf("APIMode = %q, want %q", pr.APIMode, geminiNativeAPIMode)
	}
	if pr.Path != "/models/gemini-2.5-flash:streamGenerateContent?alt=sse" {
		t.Fatalf("Path = %q", pr.Path)
	}
	var payload geminiRequest
	if err := json.Unmarshal(pr.Body, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Model != "" {
		t.Fatalf("native payload model = %q, want model only in URL path", payload.Model)
	}
	if payload.SystemInstruction == nil || payload.SystemInstruction.Parts[0].Text != "You are concise" {
		t.Fatalf("systemInstruction = %+v", payload.SystemInstruction)
	}
	if len(payload.Contents) != 3 {
		t.Fatalf("contents len = %d, want 3", len(payload.Contents))
	}
	if payload.GenerationConfig.Temperature != temp || payload.GenerationConfig.MaxOutputTokens != 512 {
		t.Fatalf("generationConfig = %+v", payload.GenerationConfig)
	}
	if len(payload.Tools) != 1 || payload.Tools[0].FunctionDeclarations[0].Name != "search" {
		t.Fatalf("tools = %+v", payload.Tools)
	}
}

func TestGeminiNativeHTTPClientUsesNativeEndpointAndAPIKeyHeader(t *testing.T) {
	var gotPath, gotAuth, gotAPIKey, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("x-goog-api-key")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"native hi\"}]}}]}\n\n"))
	}))
	defer server.Close()

	client := NewHTTPClientWithProvider(server.URL, "AIza-test", "gemini")
	status := ProviderStatusOf(client)
	if status.Runtime != geminiNativeAPIMode {
		t.Fatalf("ProviderStatus runtime = %q, want %q", status.Runtime, geminiNativeAPIMode)
	}

	stream, err := client.OpenStream(context.Background(), ChatRequest{Model: "gemini-2.5-flash", Messages: []Message{{Role: "user", Content: "Hello"}}})
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer stream.Close()
	ev, err := stream.Recv(context.Background())
	if err != nil || ev.Kind != EventToken || ev.Token != "native hi" {
		t.Fatalf("Recv = %+v err=%v", ev, err)
	}
	if gotPath != "/models/gemini-2.5-flash:streamGenerateContent?alt=sse" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAPIKey != "AIza-test" {
		t.Fatalf("x-goog-api-key = %q", gotAPIKey)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want empty for native Gemini", gotAuth)
	}
	if gotAccept != "text/event-stream" {
		t.Fatalf("Accept = %q, want text/event-stream", gotAccept)
	}
}

func TestGeminiOpenAICompatibleBaseURLDoesNotUseNativeTransport(t *testing.T) {
	client := NewHTTPClientWithProvider("https://generativelanguage.googleapis.com/v1beta/openai", "AIza-test", "gemini")
	status := ProviderStatusOf(client)
	if status.Runtime == geminiNativeAPIMode {
		t.Fatalf("ProviderStatus runtime = %q, want OpenAI-compatible Gemini route", status.Runtime)
	}
}

func TestGeminiNativeStreamNormalizesTextAndToolCalls(t *testing.T) {
	fixture := `[
		{"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]},
		{"candidates":[{"content":{"parts":[{"functionCall":{"name":"search","args":{"q":"abc"}}}]},"finishReason":"STOP"}]}
	]`
	stream, err := geminiNativeTransport{}.OpenFixtureStream(io.NopCloser(strings.NewReader(fixture)), ProviderRequest{})
	if err != nil {
		t.Fatalf("OpenFixtureStream: %v", err)
	}
	ev, err := stream.Recv(context.Background())
	if err != nil || ev.Kind != EventToken || ev.Token != "Hello" {
		t.Fatalf("first event = %+v err=%v", ev, err)
	}
	ev, err = stream.Recv(context.Background())
	if err != nil || ev.Kind != EventDone || len(ev.ToolCalls) != 1 || ev.ToolCalls[0].Name != "search" || string(ev.ToolCalls[0].Arguments) != `{"q":"abc"}` {
		t.Fatalf("second event = %+v err=%v", ev, err)
	}
}
