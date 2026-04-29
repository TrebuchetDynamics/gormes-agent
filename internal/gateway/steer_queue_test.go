package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestSteerCommandRegistry_RegisteredAsBusyAware(t *testing.T) {
	cmd, ok := ResolveCommand("/steer keep investigating")
	if !ok {
		t.Fatal("/steer did not resolve through gateway CommandRegistry")
	}
	if cmd.Name != "steer" {
		t.Fatalf("ResolveCommand(/steer).Name = %q, want steer", cmd.Name)
	}
	if cmd.Kind != EventSteer {
		t.Fatalf("/steer Kind = %v, want EventSteer", cmd.Kind)
	}
	if cmd.ActiveTurnPolicy != CommandActiveTurnPolicyDrain {
		t.Fatalf("/steer ActiveTurnPolicy = %q, want %q", cmd.ActiveTurnPolicy, CommandActiveTurnPolicyDrain)
	}
}

func TestSteerCommandRegistry_NoRunningAgentQueuesGuidance(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "active-msg")

	err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "steer-msg",
		Kind:     EventSteer,
		Text:     "/steer   keep working the selected slice   ",
	})
	if err != nil {
		t.Fatalf("handleInbound(/steer): %v", err)
	}
	if got := fk.submitsSnapshot(); len(got) != 0 {
		t.Fatalf("/steer submitted to kernel immediately: %#v", got)
	}

	next, ok := m.popNextFollowUpAsActive()
	if !ok {
		t.Fatal("/steer did not queue a follow-up event")
	}
	if next.Kind != EventSubmit {
		t.Fatalf("queued Kind = %v, want EventSubmit", next.Kind)
	}
	if next.Text != "keep working the selected slice" {
		t.Fatalf("queued Text = %q, want trimmed guidance", next.Text)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1 ack", len(sent))
	}
	for _, want := range []string{"steer_queued", "steer_preview", "keep working the selected slice"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Fatalf("ack %q missing %q", sent[0].Text, want)
		}
	}
}

func TestSteerCommandRegistry_RunningAgentFallbackDoesNotInject(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "active-msg")

	err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "steer-msg",
		Kind:     EventSteer,
		Text:     "/steer inspect the failing test before editing",
	})
	if err != nil {
		t.Fatalf("handleInbound(/steer): %v", err)
	}
	if got := fk.submitsSnapshot(); len(got) != 0 {
		t.Fatalf("running-agent fallback injected into kernel: %#v", got)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1 ack", len(sent))
	}
	for _, want := range []string{"steer_unavailable", "steer_queued", "steer_preview"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Fatalf("ack %q missing degraded evidence %q", sent[0].Text, want)
		}
	}
}
