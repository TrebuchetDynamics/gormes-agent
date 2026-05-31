package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

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
