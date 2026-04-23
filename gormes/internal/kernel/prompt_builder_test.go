package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestKernel_BuildsSecondTurnRequestWithSystemMemoryHistoryAndTools(t *testing.T) {
	rec := &mockRecall{returnContent: "<memory-context>MEMORY BLOCK</memory-context>"}
	mc := hermes.NewMockClient()

	firstAnswer := "first answer"
	firstEvents := make([]hermes.Event, 0, len(firstAnswer)+1)
	for _, ch := range firstAnswer {
		firstEvents = append(firstEvents, hermes.Event{Kind: hermes.EventToken, Token: string(ch), TokensOut: 1})
	}
	firstEvents = append(firstEvents, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"})
	mc.Script(firstEvents, "sess-prompt-builder")
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-prompt-builder")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Recall:    rec,
		Tools:     reg,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "first question"}); err != nil {
		t.Fatal(err)
	}
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		a := lastAssistantMessage(f.History)
		return f.Phase == PhaseIdle && a != nil && a.Content == firstAnswer
	}, 2*time.Second)

	if err := k.Submit(PlatformEvent{
		Kind:           PlatformEventSubmit,
		Text:           "second question",
		SessionContext: "## Current Session Context\nSource: telegram:42",
	}); err != nil {
		t.Fatal(err)
	}
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) != 2 {
		t.Fatalf("request count = %d, want 2", len(reqs))
	}

	second := reqs[1]
	if len(second.Messages) != 5 {
		t.Fatalf("len(second.Messages) = %d, want 5", len(second.Messages))
	}
	if got := second.Messages[0]; got.Role != "system" || got.Content != "## Current Session Context\nSource: telegram:42" {
		t.Fatalf("Messages[0] = %+v, want session context system message", got)
	}
	if got := second.Messages[1]; got.Role != "system" || got.Content != "<memory-context>MEMORY BLOCK</memory-context>" {
		t.Fatalf("Messages[1] = %+v, want recall system message", got)
	}
	if got := second.Messages[2]; got.Role != "user" || got.Content != "first question" {
		t.Fatalf("Messages[2] = %+v, want prior user message", got)
	}
	if got := second.Messages[3]; got.Role != "assistant" || got.Content != firstAnswer {
		t.Fatalf("Messages[3] = %+v, want prior assistant message", got)
	}
	if got := second.Messages[4]; got.Role != "user" || got.Content != "second question" {
		t.Fatalf("Messages[4] = %+v, want current user message", got)
	}

	if len(second.Tools) != 1 {
		t.Fatalf("len(second.Tools) = %d, want 1", len(second.Tools))
	}
	if second.Tools[0].Name != "echo" {
		t.Fatalf("second.Tools[0].Name = %q, want echo", second.Tools[0].Name)
	}
}
