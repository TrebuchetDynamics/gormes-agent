package main

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestHermesDialecticCaller_StreamsLLMAnswerAndSendsContextPrompt(t *testing.T) {
	client := hermes.NewMockClient()
	client.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "Use "},
		{Kind: hermes.EventToken, Token: "exact evidence."},
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "dialectic-session")

	caller := NewHermesDialecticCaller(client, "gpt-test")
	answer, err := caller.Chat(context.Background(), "telegram:6586915095", "## Peer Representation\nPrefers exact evidence.", "How should I answer?")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if answer != "Use exact evidence." {
		t.Fatalf("answer = %q, want streamed token concatenation", answer)
	}

	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", req.Model)
	}
	if !req.Stream {
		t.Fatal("dialectic caller must stream through the native Hermes client")
	}
	if len(req.Messages) != 2 {
		t.Fatalf("message count = %d, want system+user", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || !strings.Contains(req.Messages[0].Content, "Prefers exact evidence") {
		t.Fatalf("system prompt message = %+v", req.Messages[0])
	}
	if req.Messages[1].Role != "user" || req.Messages[1].Content != "How should I answer?" {
		t.Fatalf("user message = %+v", req.Messages[1])
	}
	if req.SessionID != "goncho-dialectic:telegram:6586915095" {
		t.Fatalf("session id = %q, want peer-scoped goncho dialectic session", req.SessionID)
	}
}

func TestHermesDialecticCaller_PropagatesProviderFailure(t *testing.T) {
	client := hermes.NewMockClient()

	caller := NewHermesDialecticCaller(client, "gpt-test")
	_, err := caller.Chat(context.Background(), "user", "context", "query")
	if err == nil {
		t.Fatal("Chat error = nil, want provider empty-stream error")
	}
	if !strings.Contains(err.Error(), "no dialectic answer") {
		t.Fatalf("error = %v, want no dialectic answer evidence", err)
	}
}
