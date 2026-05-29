package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestStatusSlashRendersCurrentFramePageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newStatusSlashModel(sub)
	m.frame = kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		SessionID: "sess-status-123456",
		Model:     "openai/gpt-5.3-codex",
		ProviderStatus: hermes.ProviderStatus{
			Provider: "openai-codex",
			Runtime:  "responses",
		},
		ReasoningEffort: hermes.ReasoningEffortEvidence{
			Requested: "high",
			Forwarded: true,
		},
		Telemetry: telemetry.Snapshot{
			TokensInTotal:  1200,
			TokensOutTotal: 34,
			LatencyMsLast:  987,
			TokensPerSec:   12.5,
		},
		ContextStatus: &hermes.ContextStatus{
			Engine:          "native-context",
			ContextLength:   200000,
			LastTotalTokens: 1234,
			UsagePercent:    0.617,
			Budget: hermes.ContextBudgetStatus{
				State:           "ok",
				RemainingTokens: 198766,
			},
		},
		LastError: "recoverable provider warning",
		History: []hermes.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	m = enterSlashDispatchBehavior(t, m, "/status")

	if sub.calls != 0 {
		t.Fatalf("/status reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /status = %q, want cleared", got)
	}
	if m.transientPage == nil {
		t.Fatal("/status did not open a transient page")
	}
	if m.transientPage.Title != "Status" {
		t.Fatalf("page title = %q, want Status", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range []string{
		"Gormes TUI Status",
		"Session: sess-status-123456",
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
		if !strings.Contains(body, want) {
			t.Fatalf("status page body missing %q:\n%s", want, body)
		}
	}
	view := m.View()
	if !strings.Contains(view, "Status") || !strings.Contains(view, "Gormes TUI Status") {
		t.Fatalf("View() did not render transient status page:\n%s", view)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/status fell through to fallback: %q", m.statusMessage)
	}
}

func TestStatusSlashNoActiveSessionAndBusyAvailability(t *testing.T) {
	sub := &nopSubmitter{}
	m := newStatusSlashModel(sub)
	m.frame.SessionID = ""

	m = enterSlashDispatchBehavior(t, m, "/status")
	if sub.calls != 0 {
		t.Fatalf("/status without session reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("/status without session page = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "no active session") {
		t.Fatalf("/status without session status = %q, want no active session", m.statusMessage)
	}

	completions := HermesSlashCommandCompletions("/stat")
	for _, completion := range completions {
		if completion.Name != "status" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/stat) = %+v, want status", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "status" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want status", busy)
}

func newStatusSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-status"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}
