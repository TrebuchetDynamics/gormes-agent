package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestKanbanCommandUsesInjectedRunnerAndReplies(t *testing.T) {
	ch := newFakeChannel("telegram")
	var gotInput string
	m := NewManagerWithSubmitter(ManagerConfig{
		KanbanSlashRunner: func(_ context.Context, input string) (string, error) {
			gotInput = input
			return "runner output", nil
		},
	}, &fakeKernel{}, slog.Default())

	m.handleKanbanCommand(context.Background(), ch, InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-1",
		MsgID:    "msg-1",
		Kind:     EventKanban,
		Text:     "/kanban list --status ready",
	})

	if gotInput != "/kanban list --status ready" {
		t.Fatalf("runner input = %q, want full slash input", gotInput)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1: %#v", len(sent), sent)
	}
	if sent[0].ChatID != "42" || sent[0].Text != "runner output" {
		t.Fatalf("sent = %+v, want runner output reply", sent[0])
	}
}

func TestKanbanCommandRunnerErrorIsEvidence(t *testing.T) {
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		KanbanSlashRunner: func(context.Context, string) (string, error) {
			return "partial output", errors.New("sqlite open failed")
		},
	}, &fakeKernel{}, slog.Default())

	m.handleKanbanCommand(context.Background(), ch, InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "msg-1",
		Kind:     EventKanban,
		Text:     "/kanban show missing",
	})

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1: %#v", len(sent), sent)
	}
	if !strings.Contains(sent[0].Text, "kanban error: sqlite open failed") {
		t.Fatalf("error reply = %q, want kanban error evidence", sent[0].Text)
	}
	if strings.Contains(sent[0].Text, "partial output") {
		t.Fatalf("error reply leaked partial command output: %q", sent[0].Text)
	}
}

func TestKanbanCommandLongOutputIsBoundedAndGormesBranded(t *testing.T) {
	ch := newFakeChannel("telegram")
	longOutput := strings.Repeat("x", 4200)
	m := NewManagerWithSubmitter(ManagerConfig{
		KanbanSlashRunner: func(context.Context, string) (string, error) {
			return longOutput, nil
		},
	}, &fakeKernel{}, slog.Default())

	m.handleKanbanCommand(context.Background(), ch, InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "msg-1",
		Kind:     EventKanban,
		Text:     "/kanban list",
	})

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1: %#v", len(sent), sent)
	}
	if len(sent[0].Text) > 3900 {
		t.Fatalf("bounded reply length = %d, want <= 3900", len(sent[0].Text))
	}
	if !strings.Contains(sent[0].Text, "gormes kanban") {
		t.Fatalf("truncation guidance = %q, want gormes kanban guidance", sent[0].Text)
	}
	if strings.Contains(sent[0].Text, "hermes kanban") {
		t.Fatalf("truncation guidance used stale Hermes branding: %q", sent[0].Text)
	}
}

func TestKanbanCommandBypassesActiveTurnWithoutModelLeak(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	var called int
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		KanbanSlashRunner: func(context.Context, string) (string, error) {
			called++
			return "kanban tasks", nil
		},
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "start long turn"})
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m2", Kind: EventSubmit, Text: "/kanban list"})
	waitFor(t, 200*time.Millisecond, func() bool {
		return called == 1 && len(ch.sentSnapshot()) == 1
	})

	if got := len(fk.submitsSnapshot()); got != 1 {
		t.Fatalf("kernel submits = %d, want only the original active turn", got)
	}
	for _, submit := range fk.submitsSnapshot() {
		if submit.Kind == kernel.PlatformEventSubmit && strings.Contains(submit.Text, "/kanban") {
			t.Fatalf("/kanban leaked into model submit: %+v", submit)
		}
	}
}
