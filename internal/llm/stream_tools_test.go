package llm

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

const sseToolCallsFixture = `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"echo","arguments":""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"tex"}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"t\":\"hi\"}"}}]}}]}

data: {"choices":[{"finish_reason":"tool_calls"}]}

data: [DONE]

`

func TestStream_ToolCallDeltasAccumulate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, sseToolCallsFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "x",
		Messages: []Message{{Role: "user", Content: "echo hi"}},
		Tools: []ToolDescriptor{{
			Name:        "echo",
			Description: "echo text",
			Schema:      json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var final Event
	for {
		e, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if e.Kind == EventDone {
			final = e
			break
		}
	}

	if final.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want tool_calls", final.FinishReason)
	}
	if len(final.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(final.ToolCalls))
	}
	tc := final.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ID = %q", tc.ID)
	}
	if tc.Name != "echo" {
		t.Errorf("Name = %q", tc.Name)
	}
	if !strings.Contains(string(tc.Arguments), `"hi"`) {
		t.Errorf("Arguments = %s, want to contain \"hi\"", tc.Arguments)
	}
}

// Some OpenAI-compatible providers/proxies (vLLM, llama.cpp servers, certain
// models) stream tool_call deltas but report finish_reason:"stop" instead of
// "tool_calls". The buffered calls must still be flushed onto the EventDone,
// otherwise the turn ends as plain text and the tools are silently dropped.
const sseToolCallsStopFinishFixture = `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_xyz","type":"function","function":{"name":"echo","arguments":""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"text\":\"hi\"}"}}]}}]}

data: {"choices":[{"finish_reason":"stop"}]}

data: [DONE]

`

func TestStream_ToolCallsFlushedWhenFinishReasonIsStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, sseToolCallsStopFinishFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "x",
		Messages: []Message{{Role: "user", Content: "echo hi"}},
		Tools: []ToolDescriptor{{
			Name:        "echo",
			Description: "echo text",
			Schema:      json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var final Event
	for {
		e, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if e.Kind == EventDone {
			final = e
			break
		}
	}

	if len(final.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1 (tool calls dropped on finish_reason=stop)", len(final.ToolCalls))
	}
	if final.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want tool_calls when calls are present", final.FinishReason)
	}
	if tc := final.ToolCalls[0]; tc.Name != "echo" || !strings.Contains(string(tc.Arguments), `"hi"`) {
		t.Errorf("tool call = %+v, want echo with hi", tc)
	}
}

func TestStreamDiagnosticsCapturesAllowlistedHeadersAndCounters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cf-Ray", "8f1a2b3c4d5e6f7g-LAX")
		w.Header().Set("X-Request-Id", "req-stream")
		w.Header().Set("Authorization", "Bearer should-not-leak")
		w.WriteHeader(http.StatusOK)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, `data: {"choices":[{"delta":{"content":"hello"}}]}`+"\n\n")
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "x",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	reporter, ok := s.(StreamDiagnosticsReporter)
	if !ok {
		t.Fatalf("stream does not expose StreamDiagnosticsReporter")
	}

	diag := reporter.StreamDiagnostics()
	if diag.HTTPStatus != http.StatusOK {
		t.Fatalf("HTTPStatus = %d, want %d", diag.HTTPStatus, http.StatusOK)
	}
	if got := diag.Headers["cf-ray"]; got != "8f1a2b3c4d5e6f7g-LAX" {
		t.Fatalf("cf-ray header = %q", got)
	}
	if got := diag.Headers["x-request-id"]; got != "req-stream" {
		t.Fatalf("x-request-id header = %q", got)
	}
	if _, ok := diag.Headers["authorization"]; ok {
		t.Fatalf("diagnostics leaked non-allowlisted authorization header: %#v", diag.Headers)
	}

	e, err := s.Recv(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if e.Kind != EventToken || e.Token != "hello" {
		t.Fatalf("event = %#v, want token hello", e)
	}

	diag = reporter.StreamDiagnostics()
	if diag.Chunks == 0 {
		t.Fatalf("Chunks = 0, want recorded stream chunk")
	}
	if diag.Bytes == 0 {
		t.Fatalf("Bytes = 0, want recorded stream bytes")
	}
	if diag.Elapsed <= 0 {
		t.Fatalf("Elapsed = %s, want positive duration", diag.Elapsed)
	}
	if diag.TimeToFirstByte <= 0 {
		t.Fatalf("TimeToFirstByte = %s, want positive duration", diag.TimeToFirstByte)
	}
}

func TestHTTPErrorStreamDiagnosticsSanitizesHeaders(t *testing.T) {
	err := fmt.Errorf("provider wrapper: %w", newHTTPError(http.StatusServiceUnavailable, "upstream unavailable", http.Header{
		"Cf-Ray":                []string{"8f1a2b3c4d5e6f7g-LAX"},
		"X-Openrouter-Provider": []string{"Anthropic"},
		"Authorization":         []string{"Bearer should-not-leak"},
	}))

	diag := StreamDiagnosticsFromError(err)
	if diag.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus = %d, want %d", diag.HTTPStatus, http.StatusServiceUnavailable)
	}
	if got := diag.Headers["cf-ray"]; got != "8f1a2b3c4d5e6f7g-LAX" {
		t.Fatalf("cf-ray header = %q", got)
	}
	if got := diag.Headers["x-openrouter-provider"]; got != "Anthropic" {
		t.Fatalf("x-openrouter-provider header = %q", got)
	}
	if _, ok := diag.Headers["authorization"]; ok {
		t.Fatalf("diagnostics leaked non-allowlisted authorization header: %#v", diag.Headers)
	}
}
