package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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

func TestSteerCommandRegistry_NoActiveTurnReportsUnavailable(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

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
		t.Fatalf("/steer submitted to kernel without active turn: %#v", got)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1 ack", len(sent))
	}
	for _, want := range []string{"steer_unavailable", "steer_preview", "keep working the selected slice"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Fatalf("ack %q missing %q", sent[0].Text, want)
		}
	}
}

func TestSteerCommandRegistry_RunningAgentInjectsChannelNeutralEvent(t *testing.T) {
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
	got := fk.submitsSnapshot()
	if len(got) != 1 {
		t.Fatalf("running-agent steer submits = %d, want 1", len(got))
	}
	if got[0].Kind != kernel.PlatformEventSteer || got[0].Text != "inspect the failing test before editing" {
		t.Fatalf("kernel steer event = %#v", got[0])
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1 ack", len(sent))
	}
	for _, want := range []string{"steer_injected", "steer_preview"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Fatalf("ack %q missing degraded evidence %q", sent[0].Text, want)
		}
	}
}

func TestSteerCommandRegistry_RunningAgentSubmitFailureQueuesFallback(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{submitErr: errors.New("mailbox full")}
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
