package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestIndicatorSlashControlsBusyHintWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newIndicatorSlashModel(sub)

	m = enterSlashDispatchBehavior(t, m, "/indicator unicode")
	if sub.calls != 0 {
		t.Fatalf("/indicator reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /indicator = %q, want cleared", got)
	}
	if m.indicatorStyle != IndicatorStyleUnicode {
		t.Fatalf("indicatorStyle after /indicator unicode = %q, want %q", m.indicatorStyle, IndicatorStyleUnicode)
	}
	if !strings.Contains(m.statusMessage, "indicator → unicode") {
		t.Fatalf("status after /indicator unicode = %q, want indicator evidence", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/indicator fell through to fallback: %q", m.statusMessage)
	}

	hint := m.renderHermesHint()
	if !strings.Contains(hint, "⠋") {
		t.Fatalf("unicode indicator hint = %q, want braille frame", hint)
	}
	if strings.Contains(hint, "◕") || strings.Contains(hint, "⚕") {
		t.Fatalf("unicode indicator hint leaked another style: %q", hint)
	}

	m = enterSlashDispatchBehavior(t, m, "/indicator")
	if !strings.Contains(m.statusMessage, "indicator: unicode") {
		t.Fatalf("bare /indicator status = %q, want current style", m.statusMessage)
	}
}

func TestIndicatorSlashUsageAndCompletions(t *testing.T) {
	m := newIndicatorSlashModel(&nopSubmitter{})
	m.indicatorStyle = IndicatorStyleEmoji

	m = enterSlashDispatchBehavior(t, m, "/indicator sparkle")
	if m.indicatorStyle != IndicatorStyleEmoji {
		t.Fatalf("invalid /indicator changed style to %q, want %q", m.indicatorStyle, IndicatorStyleEmoji)
	}
	if !strings.Contains(m.statusMessage, "usage: /indicator [ascii|emoji|kaomoji|unicode]") {
		t.Fatalf("invalid /indicator status = %q, want usage", m.statusMessage)
	}

	completions := HermesSlashCommandCompletions("/ind")
	for _, completion := range completions {
		if completion.Name != "indicator" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/ind) = %+v, want indicator", completions)

foundCompletion:
	got := completionNames(HermesSlashSubcommandCompletions("/indicator "))
	want := []string{"ascii", "emoji", "kaomoji", "unicode"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("HermesSlashSubcommandCompletions(/indicator ) = %v, want %v", got, want)
	}

	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "indicator" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want indicator", busy)
}

func newIndicatorSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseStreaming, SessionID: "sess-indicator"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}
