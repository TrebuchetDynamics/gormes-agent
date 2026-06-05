package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type capturedProxyRequest struct {
	Path          string
	Authorization string
	SessionID     string
	Messages      []map[string]any
	Stream        bool
}

func TestProxySubmitter_ForwardsSessionHeaderAndFiltersUnsafeHistory(t *testing.T) {
	requests := make(chan capturedProxyRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]any `json:"messages"`
			Stream   bool             `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		requests <- capturedProxyRequest{
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
			SessionID:     r.Header.Get("X-Hermes-Session-Id"),
			Messages:      body.Messages,
			Stream:        body.Stream,
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Hermes-Session-Id", r.Header.Get("X-Hermes-Session-Id"))
		fmt.Fprint(w, `data: {"choices":[{"delta":{"role":"assistant"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"Hello"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":" world"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"finish_reason":"stop","delta":{}}]}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	proxy, err := NewProxySubmitter(ProxySubmitterConfig{
		BaseURL: srv.URL + "/",
		APIKey:  "secret-key",
		Model:   "gormes-agent",
		History: []llm.Message{
			{Role: "user", Content: "previous user"},
			{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "search"}}},
			{Role: "tool", Content: "tool result", ToolCallID: "call_1", Name: "search"},
			{Role: "assistant", Content: "  "},
			{Role: "assistant", Content: "previous assistant", ToolCalls: []llm.ToolCall{{ID: "call_2", Name: "ignored"}}},
		},
	})
	if err != nil {
		t.Fatalf("NewProxySubmitter: %v", err)
	}

	err = proxy.Submit(kernel.PlatformEvent{
		Kind:           kernel.PlatformEventSubmit,
		Text:           "tell me more",
		SessionID:      "sess-abc",
		SessionContext: "## Current Session Context\nplatform: matrix",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	var req capturedProxyRequest
	select {
	case req = <-requests:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("proxy request not received")
	}
	if req.Path != "/v1/chat/completions" {
		t.Fatalf("path = %q, want /v1/chat/completions", req.Path)
	}
	if req.Authorization != "Bearer secret-key" {
		t.Fatalf("Authorization = %q, want bearer key", req.Authorization)
	}
	if req.SessionID != "sess-abc" {
		t.Fatalf("X-Hermes-Session-Id = %q, want sess-abc", req.SessionID)
	}
	if !req.Stream {
		t.Fatal("stream = false, want true")
	}

	want := []map[string]string{
		{"role": "system", "content": "## Current Session Context\nplatform: matrix"},
		{"role": "user", "content": "previous user"},
		{"role": "assistant", "content": "previous assistant"},
		{"role": "user", "content": "tell me more"},
	}
	if len(req.Messages) != len(want) {
		t.Fatalf("messages len = %d, want %d: %#v", len(req.Messages), len(want), req.Messages)
	}
	for i, msg := range req.Messages {
		if msg["role"] != want[i]["role"] || msg["content"] != want[i]["content"] {
			t.Fatalf("messages[%d] = %#v, want role/content %#v", i, msg, want[i])
		}
		if _, ok := msg["tool_calls"]; ok {
			t.Fatalf("messages[%d] forwarded tool_calls: %#v", i, msg)
		}
		if _, ok := msg["tool_call_id"]; ok {
			t.Fatalf("messages[%d] forwarded tool_call_id: %#v", i, msg)
		}
	}

	final := readProxyTerminalFrame(t, proxy.Render())
	if final.Phase != kernel.PhaseIdle {
		t.Fatalf("terminal phase = %v, want idle", final.Phase)
	}
	if final.SessionID != "sess-abc" {
		t.Fatalf("terminal SessionID = %q, want sess-abc", final.SessionID)
	}
	if got := final.History[len(final.History)-1].Content; got != "Hello world" {
		t.Fatalf("final assistant = %q, want streamed content", got)
	}
}

func TestProxySubmitter_PreservesAssistantReplayMetadata(t *testing.T) {
	requests := make(chan capturedProxyRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]any `json:"messages"`
			Stream   bool             `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		requests <- capturedProxyRequest{Messages: body.Messages, Stream: body.Stream}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"metadata ok"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"finish_reason":"stop","delta":{}}]}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	structuredReasoning := "structured provider reasoning"
	emptyReasoning := ""
	proxy, err := NewProxySubmitter(ProxySubmitterConfig{
		BaseURL: srv.URL,
		Model:   "deepseek/deepseek-reasoner",
		History: []llm.Message{
			{Role: "assistant", Content: "reasoned answer", Reasoning: &llm.ReasoningContent{Text: "storage-only reasoning"}, ReasoningContent: &structuredReasoning},
			{Role: "assistant", Content: "empty sentinel", ReasoningContent: &emptyReasoning},
			{Role: "assistant", ContentParts: []llm.MessageContentPart{{Type: "text", Text: "content part survived"}}},
			{Role: "assistant", Content: "cached answer", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
			{Role: "assistant", Content: "unsafe tool-call carrier", ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "lookup"}}},
			{Role: "tool", Content: "tool result", ToolCallID: "call_1", Name: "lookup"},
		},
	})
	if err != nil {
		t.Fatalf("NewProxySubmitter: %v", err)
	}

	if err := proxy.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "continue", SessionID: "sess-replay"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	var req capturedProxyRequest
	select {
	case req = <-requests:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("proxy request not received")
	}
	if !req.Stream {
		t.Fatal("stream = false, want true")
	}
	reasoned := proxyMessageByContent(t, req.Messages, "reasoned answer")
	if got := reasoned["reasoning_content"]; got != structuredReasoning {
		t.Fatalf("reasoned reasoning_content = %#v, want %q", got, structuredReasoning)
	}
	sentinel := proxyMessageByContent(t, req.Messages, "empty sentinel")
	if got, ok := sentinel["reasoning_content"].(string); !ok || got != "" {
		t.Fatalf("sentinel reasoning_content = %#v (present=%v), want empty string", sentinel["reasoning_content"], ok)
	}
	parts := proxyMessageByRoleAndContentPart(t, req.Messages, "assistant", "content part survived")
	if _, ok := parts["tool_calls"]; ok {
		t.Fatalf("content-parts message leaked tool_calls: %#v", parts)
	}
	unsafe := proxyMessageByContent(t, req.Messages, "unsafe tool-call carrier")
	if _, ok := unsafe["tool_calls"]; ok {
		t.Fatalf("unsafe tool-call carrier forwarded tool_calls: %#v", unsafe)
	}
	for _, msg := range req.Messages {
		if msg["role"] == "tool" || msg["tool_call_id"] != nil {
			t.Fatalf("proxy replay forwarded unsafe tool history: %#v", msg)
		}
	}

	final := readProxyTerminalFrame(t, proxy.Render())
	var foundStructuredReasoning bool
	var foundEmptySentinel bool
	var foundCacheControl bool
	for _, msg := range final.History {
		if msg.Content == "reasoned answer" {
			foundStructuredReasoning = msg.Reasoning != nil &&
				msg.Reasoning.Text == "storage-only reasoning" &&
				msg.ReasoningContent != nil &&
				*msg.ReasoningContent == structuredReasoning
		}
		if msg.Content == "empty sentinel" {
			foundEmptySentinel = msg.ReasoningContent != nil && *msg.ReasoningContent == ""
		}
		if msg.Content == "cached answer" {
			foundCacheControl = msg.CacheControl != nil && msg.CacheControl.Type == "ephemeral"
		}
	}
	if !foundStructuredReasoning {
		t.Fatalf("final history did not preserve structured assistant reasoning: %#v", final.History)
	}
	if !foundEmptySentinel {
		t.Fatalf("final history did not preserve empty reasoning_content sentinel: %#v", final.History)
	}
	if !foundCacheControl {
		t.Fatalf("final history did not preserve assistant cache control: %#v", final.History)
	}
}

func TestProxySubmitter_PreservesAssistantReplayMetadataForClientRequest(t *testing.T) {
	client := llm.NewMockClient()
	client.Script([]llm.Event{{Kind: llm.EventToken, Token: "client metadata ok"}}, "sess-client-replay")

	structuredReasoning := "structured provider reasoning"
	emptyReasoning := ""
	proxy, err := NewProxySubmitter(ProxySubmitterConfig{
		Client: client,
		Model:  "deepseek/deepseek-reasoner",
		History: []llm.Message{
			{Role: "assistant", Content: "reasoned answer", Reasoning: &llm.ReasoningContent{Text: "storage-only reasoning"}, ReasoningContent: &structuredReasoning},
			{Role: "assistant", Content: "empty sentinel", ReasoningContent: &emptyReasoning},
			{Role: "assistant", ContentParts: []llm.MessageContentPart{{Type: "text", Text: "content part survived"}}},
			{Role: "assistant", Content: "cached answer", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
			{Role: "assistant", Content: "unsafe tool-call carrier", ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "lookup"}}},
			{Role: "tool", Content: "tool result", ToolCallID: "call_1", Name: "lookup"},
		},
	})
	if err != nil {
		t.Fatalf("NewProxySubmitter: %v", err)
	}

	if err := proxy.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "continue", SessionID: "sess-replay"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	final := readProxyTerminalFrame(t, proxy.Render())
	if final.SessionID != "sess-client-replay" {
		t.Fatalf("terminal SessionID = %q, want remote session", final.SessionID)
	}

	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("client requests len = %d, want 1", len(requests))
	}
	reasoned := proxyClientMessageByContent(t, requests[0].Messages, "reasoned answer")
	if reasoned.Reasoning == nil || reasoned.Reasoning.Text != "storage-only reasoning" {
		t.Fatalf("reasoned Reasoning = %#v, want storage-only reasoning", reasoned.Reasoning)
	}
	if reasoned.ReasoningContent == nil || *reasoned.ReasoningContent != structuredReasoning {
		t.Fatalf("reasoned ReasoningContent = %#v, want %q", reasoned.ReasoningContent, structuredReasoning)
	}
	sentinel := proxyClientMessageByContent(t, requests[0].Messages, "empty sentinel")
	if sentinel.ReasoningContent == nil || *sentinel.ReasoningContent != "" {
		t.Fatalf("sentinel ReasoningContent = %#v, want empty string pointer", sentinel.ReasoningContent)
	}
	parts := proxyClientMessageByContentPart(t, requests[0].Messages, "content part survived")
	if len(parts.ContentParts) != 1 || parts.ContentParts[0].Text != "content part survived" {
		t.Fatalf("parts ContentParts = %#v, want preserved text part", parts.ContentParts)
	}
	cached := proxyClientMessageByContent(t, requests[0].Messages, "cached answer")
	if cached.CacheControl == nil || cached.CacheControl.Type != "ephemeral" {
		t.Fatalf("cached CacheControl = %#v, want ephemeral", cached.CacheControl)
	}
	unsafe := proxyClientMessageByContent(t, requests[0].Messages, "unsafe tool-call carrier")
	if len(unsafe.ToolCalls) != 0 {
		t.Fatalf("unsafe carrier forwarded tool calls: %#v", unsafe.ToolCalls)
	}
	for _, msg := range requests[0].Messages {
		if msg.Role == "tool" || msg.ToolCallID != "" || msg.Name != "" {
			t.Fatalf("proxy replay forwarded unsafe tool history: %#v", msg)
		}
	}
}

func TestProxySubmitter_StaleGenerationReportsDegradedOutput(t *testing.T) {
	requestSeen := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestSeen)
		<-release
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Hermes-Session-Id", r.Header.Get("X-Hermes-Session-Id"))
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"stale answer"}}]}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	store := &capturingRuntimeStatus{}
	proxy, err := NewProxySubmitter(ProxySubmitterConfig{
		BaseURL:       srv.URL,
		Model:         "gormes-agent",
		RuntimeStatus: store,
	})
	if err != nil {
		t.Fatalf("NewProxySubmitter: %v", err)
	}
	if err := proxy.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "hi", SessionID: "sess-stale"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	select {
	case <-requestSeen:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("proxy request not received")
	}
	if err := proxy.ResetSession(); err != nil {
		t.Fatalf("ResetSession: %v", err)
	}
	close(release)

	final := readProxyTerminalFrame(t, proxy.Render())
	if final.Phase != kernel.PhaseFailed {
		t.Fatalf("terminal phase = %v, want failed", final.Phase)
	}
	if !strings.Contains(final.LastError, "stale generation") {
		t.Fatalf("LastError = %q, want stale generation degradation", final.LastError)
	}
	for _, msg := range final.History {
		if strings.Contains(msg.Content, "stale answer") {
			t.Fatalf("stale remote content was accepted into history: %#v", final.History)
		}
	}
	assertProxyStatus(t, store, "degraded", "stale generation")
}

func TestProxySubmitter_RemoteErrorsReturnVisibleDegradedOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
	}))
	defer srv.Close()

	store := &capturingRuntimeStatus{}
	proxy, err := NewProxySubmitter(ProxySubmitterConfig{
		BaseURL:       srv.URL,
		Model:         "gormes-agent",
		RuntimeStatus: store,
	})
	if err != nil {
		t.Fatalf("NewProxySubmitter: %v", err)
	}
	if err := proxy.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "hi", SessionID: "sess-auth"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	final := readProxyTerminalFrame(t, proxy.Render())
	if final.Phase != kernel.PhaseFailed {
		t.Fatalf("terminal phase = %v, want failed", final.Phase)
	}
	if !strings.Contains(final.LastError, "missing proxy credentials") {
		t.Fatalf("LastError = %q, want missing proxy credentials degradation", final.LastError)
	}
	assertProxyStatus(t, store, "degraded", "missing proxy credentials")
}

func TestProxySubmitter_UnreachableProxyReportsDegradedOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "unused")
	}))
	baseURL := srv.URL
	srv.Close()

	store := &capturingRuntimeStatus{}
	proxy, err := NewProxySubmitter(ProxySubmitterConfig{
		BaseURL:       baseURL,
		Model:         "gormes-agent",
		APIKey:        "secret",
		RuntimeStatus: store,
	})
	if err != nil {
		t.Fatalf("NewProxySubmitter: %v", err)
	}
	if err := proxy.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "hi", SessionID: "sess-down"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	final := readProxyTerminalFrame(t, proxy.Render())
	if final.Phase != kernel.PhaseFailed {
		t.Fatalf("terminal phase = %v, want failed", final.Phase)
	}
	if !strings.Contains(final.LastError, "proxy unreachable") {
		t.Fatalf("LastError = %q, want proxy unreachable degradation", final.LastError)
	}
	assertProxyStatus(t, store, "degraded", "proxy unreachable")
}

func proxyMessageByContent(t *testing.T, messages []map[string]any, content string) map[string]any {
	t.Helper()
	for _, msg := range messages {
		if msg["content"] == content {
			return msg
		}
	}
	t.Fatalf("message with content %q not found in %#v", content, messages)
	return nil
}

func proxyMessageByRoleAndContentPart(t *testing.T, messages []map[string]any, role, text string) map[string]any {
	t.Helper()
	for _, msg := range messages {
		if msg["role"] == role && proxyContentPartContainsText(msg, text) {
			return msg
		}
	}
	t.Fatalf("%s message with content part %q not found in %#v", role, text, messages)
	return nil
}

func proxyClientMessageByContent(t *testing.T, messages []llm.Message, content string) llm.Message {
	t.Helper()
	for _, msg := range messages {
		if msg.Content == content {
			return msg
		}
	}
	t.Fatalf("client message with content %q not found in %#v", content, messages)
	return llm.Message{}
}

func proxyClientMessageByContentPart(t *testing.T, messages []llm.Message, text string) llm.Message {
	t.Helper()
	for _, msg := range messages {
		for _, part := range msg.ContentParts {
			if part.Text == text {
				return msg
			}
		}
	}
	t.Fatalf("client message with content part %q not found in %#v", text, messages)
	return llm.Message{}
}

func proxyContentPartContainsText(msg map[string]any, text string) bool {
	parts, ok := msg["content"].([]any)
	if !ok {
		return false
	}
	for _, part := range parts {
		partMap, ok := part.(map[string]any)
		if ok && partMap["text"] == text {
			return true
		}
	}
	return false
}

func readProxyTerminalFrame(t *testing.T, frames <-chan kernel.RenderFrame) kernel.RenderFrame {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case f := <-frames:
			if f.Phase == kernel.PhaseIdle || f.Phase == kernel.PhaseFailed || f.Phase == kernel.PhaseCancelling {
				return f
			}
		case <-timeout:
			t.Fatal("timed out waiting for proxy terminal frame")
		}
	}
}

type capturingRuntimeStatus struct {
	mu     sync.Mutex
	status RuntimeStatusUpdate
}

func (s *capturingRuntimeStatus) UpdateRuntimeStatus(ctx context.Context, update RuntimeStatusUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = update
	return nil
}

func (s *capturingRuntimeStatus) snapshot() RuntimeStatusUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func assertProxyStatus(t *testing.T, store *capturingRuntimeStatus, wantState, wantMessage string) {
	t.Helper()
	status := store.snapshot()
	if status.ProxyState != wantState {
		t.Fatalf("proxy status = %q, want %q", status.ProxyState, wantState)
	}
	if !strings.Contains(status.ProxyErrorMessage, wantMessage) {
		t.Fatalf("proxy error = %q, want %q", status.ProxyErrorMessage, wantMessage)
	}
}
