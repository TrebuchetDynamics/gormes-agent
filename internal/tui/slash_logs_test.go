package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestLogsSlashRendersGatewayTailPageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var gotLimit int
	m := newLogsSlashModel(sub, func(limit int) (string, error) {
		gotLimit = limit
		return "gateway line one\ngateway line two", nil
	})

	m = enterSlashDispatchBehavior(t, m, "/logs 500")

	if sub.calls != 0 {
		t.Fatalf("/logs reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /logs = %q, want cleared", got)
	}
	if gotLimit != 80 {
		t.Fatalf("/logs 500 requested limit %d, want Hermes clamp to 80", gotLimit)
	}
	if m.transientPage == nil {
		t.Fatal("/logs did not open a transient page")
	}
	if m.transientPage.Title != "Logs" {
		t.Fatalf("page title = %q, want Logs", m.transientPage.Title)
	}
	for _, want := range []string{"gateway line one", "gateway line two"} {
		if !strings.Contains(m.transientPage.Body, want) {
			t.Fatalf("logs page body missing %q:\n%s", want, m.transientPage.Body)
		}
	}
	view := m.View()
	if !strings.Contains(view, "Logs") || !strings.Contains(view, "gateway line one") {
		t.Fatalf("View() did not render transient logs page:\n%s", view)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/logs fell through to fallback: %q", m.statusMessage)
	}
}

func TestLogsSlashNoLogsAndBusyAvailability(t *testing.T) {
	sub := &nopSubmitter{}
	m := newLogsSlashModel(sub, func(limit int) (string, error) {
		if limit != 20 {
			t.Fatalf("/logs default limit = %d, want 20", limit)
		}
		return "", nil
	})

	m = enterSlashDispatchBehavior(t, m, "/logs 0")
	if sub.calls != 0 {
		t.Fatalf("/logs with empty tail reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("/logs with empty tail page = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "no gateway logs") {
		t.Fatalf("/logs empty status = %q, want no gateway logs", m.statusMessage)
	}

	completions := HermesSlashCommandCompletions("/lo")
	for _, completion := range completions {
		if completion.Name != "logs" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/lo) = %+v, want logs", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "logs" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want logs", busy)
}

func newLogsSlashModel(sub *nopSubmitter, tail GatewayLogTailFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-logs"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, GatewayLogTail: tail})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}
