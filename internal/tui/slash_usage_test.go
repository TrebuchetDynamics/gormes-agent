package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestUsageSlashOpensFrameUsagePageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newUsageSlashModel(sub, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: "sess-usage",
		Model:     "gpt-usage",
		Telemetry: telemetry.Snapshot{
			TokensInTotal:  1234,
			TokensOutTotal: 567,
			LatencyMsLast:  890,
			TokensPerSec:   12.34,
		},
		ContextStatus: &llm.ContextStatus{
			ContextLength:    200000,
			LastTotalTokens:  18000,
			UsagePercent:     9.0,
			CompressionCount: 2,
		},
	})

	m = enterSlashDispatchBehavior(t, m, "/usage")
	if sub.calls != 0 {
		t.Fatalf("/usage reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /usage = %q, want cleared", got)
	}
	if !strings.Contains(m.statusMessage, "usage opened") {
		t.Fatalf("/usage status = %q, want usage opened", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/usage fell through to fallback: %q", m.statusMessage)
	}
	assertUsagePageContains(t, m,
		"Usage source: local TUI frame",
		"Model: gpt-usage",
		"Session: sess-usage",
		"Input tokens: 1234",
		"Output tokens: 567",
		"Total tokens: 1801",
		"Last latency: 890 ms",
		"Speed: 12.34 tokens/sec",
		"Context: 18000 / 200000 tokens (9.0%)",
		"Compressions: 2",
	)
}

func TestUsageSlashFetchesAccountUsageAsynchronously(t *testing.T) {
	sub := &nopSubmitter{}
	used := 40.0
	var requested bool
	m := newUsageSlashModelWithOptions(sub, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: "sess-usage",
		Model:     "gpt-usage",
		Telemetry: telemetry.Snapshot{TokensInTotal: 10, TokensOutTotal: 5},
	}, Options{
		MouseTracking: true,
		AccountUsage: func(ctx context.Context) (llm.AccountUsageSnapshot, error) {
			if ctx == nil {
				t.Fatal("AccountUsage context is nil")
			}
			requested = true
			return llm.AccountUsageSnapshot{
				Provider:  "openrouter",
				Plan:      "Team",
				FetchedAt: time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC),
				Windows: []llm.AccountUsageWindow{{
					Label:       "Session",
					UsedPercent: &used,
				}},
				Details: []string{"Credits balance: $12.50"},
			}, nil
		},
	})

	m.editor.SetValue("/usage")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = requireUsageSlashModel(t, next)
	if sub.calls != 0 {
		t.Fatalf("/usage reached Submitter %d time(s), want 0", sub.calls)
	}
	if requested {
		t.Fatal("AccountUsage was called before the returned tea.Cmd ran")
	}
	assertUsagePageContains(t, m, "Provider account usage: loading...")

	msg := firstNonNilCmdMsg(t, cmd)
	if msg == nil {
		t.Fatal("/usage account command returned nil message, want usage account update")
	}
	next, _ = m.Update(msg)
	m = requireUsageSlashModel(t, next)
	if !requested {
		t.Fatal("AccountUsage was not called by returned tea.Cmd")
	}
	if !strings.Contains(m.statusMessage, "usage account updated") {
		t.Fatalf("status after account update = %q, want usage account updated", m.statusMessage)
	}
	assertUsagePageContains(t, m,
		"Provider: openrouter (Team)",
		"Session: 60% remaining (40% used)",
		"Credits balance: $12.50",
	)
}

func TestUsageSlashNoCallsConsumesWithoutPage(t *testing.T) {
	sub := &nopSubmitter{}
	m := newUsageSlashModel(sub, kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-empty", Model: "gpt-empty"})

	m = enterSlashDispatchBehavior(t, m, "/usage")
	if sub.calls != 0 {
		t.Fatalf("/usage with no counters reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "no API calls yet") {
		t.Fatalf("/usage no counters status = %q, want no API calls yet", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("/usage no counters opened page %+v, want nil", *m.transientPage)
	}
}

func TestUsageSlashCompletionsAndBusyAvailability(t *testing.T) {
	for _, completion := range HermesSlashCommandCompletions("/usa") {
		if completion.Name != "usage" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/usa) missing usage")

foundCompletion:
	for _, name := range NewDefaultSlashRegistry().BusyAvailableSlashes() {
		if name == "usage" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() missing usage")
}

func assertUsagePageContains(t *testing.T, m Model, want ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("/usage did not open a transient page")
	}
	if m.transientPage.Title != "Usage" {
		t.Fatalf("page title = %q, want Usage", m.transientPage.Title)
	}
	for _, item := range want {
		if !strings.Contains(m.transientPage.Body, item) {
			t.Fatalf("usage page missing %q:\n%s", item, m.transientPage.Body)
		}
	}
}

func newUsageSlashModel(sub *nopSubmitter, frame kernel.RenderFrame) Model {
	return newUsageSlashModelWithOptions(sub, frame, Options{MouseTracking: true})
}

func newUsageSlashModelWithOptions(sub *nopSubmitter, frame kernel.RenderFrame, opts Options) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- frame
	if !opts.MouseTracking {
		opts.MouseTracking = true
	}
	m := NewModelWithOptions(frames, sub.submit, func() {}, opts)
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

func requireUsageSlashModel(t *testing.T, model tea.Model) Model {
	t.Helper()
	m, ok := model.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", model)
	}
	return m
}

func firstNonNilCmdMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, nested := range batch {
			if got := firstNonNilCmdMsg(t, nested); got != nil {
				return got
			}
		}
		return nil
	}
	return msg
}
