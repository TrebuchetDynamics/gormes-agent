package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestKernel_InjectsPrefillMessagesAfterSystemBeforeUser(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-prefill")

	k := New(Config{
		Model:     "gpt-5",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		PrefillMessages: []hermes.Message{
			{Role: "user", Content: "example request"},
			{Role: "assistant", Content: "example answer"},
		},
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "real request"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	messages := reqs[0].Messages
	if len(messages) < 4 {
		t.Fatalf("Messages = %#v, want guidance + two prefill messages + current user", messages)
	}
	if messages[0].Role != "system" {
		t.Fatalf("Messages[0].Role = %q, want leading system guidance", messages[0].Role)
	}
	firstNonSystem := 0
	for firstNonSystem < len(messages) && messages[firstNonSystem].Role == "system" {
		firstNonSystem++
	}
	if firstNonSystem == 0 || firstNonSystem+2 >= len(messages) {
		t.Fatalf("Messages = %#v, want system messages followed by prefill and current user", messages)
	}
	if messages[firstNonSystem].Role != "user" || messages[firstNonSystem].Content != "example request" {
		t.Fatalf("first non-system message = %#v, want first prefill user message", messages[firstNonSystem])
	}
	if messages[firstNonSystem+1].Role != "assistant" || messages[firstNonSystem+1].Content != "example answer" {
		t.Fatalf("message after first prefill = %#v, want second prefill assistant message", messages[firstNonSystem+1])
	}
	if last := messages[len(messages)-1]; last.Role != "user" || last.Content != "real request" {
		t.Fatalf("last message = %#v, want current user message", last)
	}
}

func TestKernel_PrefillMessagesAreNotPersistedInVisibleHistory(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventToken, Token: "done"}, {Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-prefill-history")

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		PrefillMessages: []hermes.Message{
			{Role: "user", Content: "hidden example"},
			{Role: "assistant", Content: "hidden answer"},
		},
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "visible user"})

	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)
	if len(final.History) != 2 {
		t.Fatalf("History = %#v, want only visible user + assistant", final.History)
	}
	if final.History[0].Content != "visible user" || final.History[1].Content != "done" {
		t.Fatalf("History = %#v, want prefill omitted", final.History)
	}
}
