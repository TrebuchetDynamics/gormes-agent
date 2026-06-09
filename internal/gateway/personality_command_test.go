package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestPersonalityCommandSanitizesSetConfirmation(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())
	m.personalityPrompts = map[string]string{"bad**name**": "prompt"}
	ch := newFakeChannel("telegram")

	handled, err := m.dispatchGatewayCommandEvent(context.Background(), ch, InboundEvent{
		Kind:     EventPersonality,
		Text:     "/personality bad**name**",
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "m1",
	})
	if err != nil || !handled {
		t.Fatalf("dispatchGatewayCommandEvent handled=%v err=%v", handled, err)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	if strings.Contains(sent[0].Text, "**name**") || !strings.Contains(sent[0].Text, "Personality set to **bad''name''**.") {
		t.Fatalf("personality confirmation not sanitized: %#v", sent)
	}
}

func TestPersonalityCommandPreservesParsedArgument(t *testing.T) {
	kind, body := ParseInboundText("/personality pirate")
	if kind != EventPersonality || body != "/personality pirate" {
		t.Fatalf("ParseInboundText(/personality pirate) = (%v, %q), want EventPersonality with raw body", kind, body)
	}

	m := NewManager(ManagerConfig{}, nil, slog.Default())
	m.personalityPrompts = map[string]string{"pirate": "talk like a pirate"}
	ch := newFakeChannel("telegram")

	handled, err := m.dispatchGatewayCommandEvent(context.Background(), ch, InboundEvent{
		Kind:     kind,
		Text:     body,
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "m1",
	})
	if err != nil || !handled {
		t.Fatalf("dispatchGatewayCommandEvent handled=%v err=%v", handled, err)
	}
	if got := m.activePersonality(); got != "pirate" {
		t.Fatalf("active personality = %q, want pirate", got)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Personality set to **pirate**") {
		t.Fatalf("sent = %#v, want personality confirmation", sent)
	}
}
