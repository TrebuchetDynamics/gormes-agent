package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestManagerQueueCommandQueuesFIFOAndDoesNotLeak(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "active-msg")

	for _, input := range []string{"/queue first follow-up", "/q second follow-up"} {
		if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: input, Kind: EventSubmit, Text: input}); err != nil {
			t.Fatalf("handleInbound(%q): %v", input, err)
		}
	}
	if got := fk.submitsSnapshot(); len(got) != 0 {
		t.Fatalf("/queue submitted before active turn finished: %#v", got)
	}
	assertSentContains(t, ch.sentSnapshot(), "Queued for the next turn.")
	assertSentContains(t, ch.sentSnapshot(), "(2 queued)")

	var co *coalescer
	var cancel context.CancelFunc
	m.dispatchFrame(context.Background(), kernel.RenderFrame{Phase: kernel.PhaseIdle, DraftText: "active done"}, &co, &cancel)
	got := fk.submitsSnapshot()
	if len(got) != 1 || got[0].Text != "first follow-up" || got[0].Kind != kernel.PlatformEventSubmit {
		t.Fatalf("first queued submit = %#v, want first follow-up", got)
	}
	if strings.HasPrefix(strings.TrimSpace(got[0].Text), "/") {
		t.Fatalf("raw slash leaked into first queued submit: %#v", got[0])
	}

	m.dispatchFrame(context.Background(), kernel.RenderFrame{Phase: kernel.PhaseIdle, DraftText: "first follow-up done"}, &co, &cancel)
	got = fk.submitsSnapshot()
	if len(got) != 2 || got[1].Text != "second follow-up" || got[1].Kind != kernel.PlatformEventSubmit {
		t.Fatalf("queued submits = %#v, want second follow-up after first", got)
	}
	if strings.HasPrefix(strings.TrimSpace(got[1].Text), "/") {
		t.Fatalf("raw slash leaked into second queued submit: %#v", got[1])
	}
}

func TestManagerQueueCommandUsageAndNoActiveTurnDegrades(t *testing.T) {
	ch := newFakeChannel("discord")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"discord": "C42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "/queue"}); err != nil {
		t.Fatalf("handleInbound(/queue): %v", err)
	}
	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m2", Kind: EventSubmit, Text: "/queue run later"}); err != nil {
		t.Fatalf("handleInbound(/queue run later): %v", err)
	}

	sent := ch.sentSnapshot()
	assertSentContains(t, sent, "Usage: /queue <prompt>")
	assertSentContains(t, sent, "queue_unavailable")
	if got := fk.submitsSnapshot(); len(got) != 0 {
		t.Fatalf("queue degraded path submitted to kernel: %#v", got)
	}
}

func assertSentContains(t *testing.T, sent []fakeSent, want string) {
	t.Helper()
	for _, msg := range sent {
		if strings.Contains(msg.Text, want) {
			return
		}
	}
	t.Fatalf("sent messages missing %q: %#v", want, sent)
}
