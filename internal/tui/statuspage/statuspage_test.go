package statuspage

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestBuildStatusPageFormatsFrameEvidence(t *testing.T) {
	page := Build(kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		SessionID: "frame-session",
		Model:     "openai/gpt-5.3-codex",
		ProviderStatus: llm.ProviderStatus{
			Provider: "openai-codex",
			Runtime:  "responses",
		},
		ReasoningEffort: llm.ReasoningEffortEvidence{Requested: "high", Forwarded: true},
		Telemetry: telemetry.Snapshot{
			TokensInTotal:  1200,
			TokensOutTotal: 34,
			LatencyMsLast:  987,
			TokensPerSec:   12.5,
		},
		ContextStatus: &llm.ContextStatus{
			Engine:          "native-context",
			ContextLength:   200000,
			LastTotalTokens: 1234,
			UsagePercent:    0.617,
			Budget: llm.ContextBudgetStatus{
				State:           "ok",
				RemainingTokens: 198766,
			},
		},
		LastError: "recoverable provider warning",
		History: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}, "explicit-session")

	if page.Title != "Status" {
		t.Fatalf("title = %q, want Status", page.Title)
	}
	for _, want := range []string{
		"Gormes TUI Status",
		"Session: explicit-session",
		"Phase: Streaming",
		"Model: openai/gpt-5.3-codex",
		"Provider: openai-codex",
		"Runtime: responses",
		"Reasoning effort: high (forwarded)",
		"Context: 1234 / 200000 tokens",
		"Budget: ok, 198766 tokens remaining",
		"Telemetry: 1200 in / 34 out / 987 ms / 12.5 tok/s",
		"History messages: 2",
		"Last error: recoverable provider warning",
	} {
		if !strings.Contains(page.Body, want) {
			t.Fatalf("body missing %q:\n%s", want, page.Body)
		}
	}
}

func TestBuildStatusPageFallsBackToFrameSession(t *testing.T) {
	page := Build(kernel.RenderFrame{SessionID: "frame-session"}, "  ")
	if !strings.Contains(page.Body, "Session: frame-session") {
		t.Fatalf("body missing frame session fallback:\n%s", page.Body)
	}
}

func TestHandleSlashRequiresActiveSessionAndOpensStatusPage(t *testing.T) {
	closed := HandleSlash(kernel.RenderFrame{SessionID: "frame-session"}, "  ")
	if closed.Open || closed.StatusMessage != "no active session" {
		t.Fatalf("closed /status result = %+v, want no active session", closed)
	}

	opened := HandleSlash(kernel.RenderFrame{Phase: kernel.PhaseIdle}, "sess-status")
	if !opened.Open || opened.StatusMessage != "status opened" || opened.Page.Title != "Status" {
		t.Fatalf("opened /status result = %+v", opened)
	}
	if !strings.Contains(opened.Page.Body, "Session: sess-status") {
		t.Fatalf("opened page missing session:\n%s", opened.Page.Body)
	}
}
