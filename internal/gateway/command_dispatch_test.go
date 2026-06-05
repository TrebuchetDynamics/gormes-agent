package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestGatewayCommandDispatchHandlesDirectAndSlashBusyCommand(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   InboundEvent
	}{
		{name: "direct event", ev: InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventBusy, Text: "/busy queue"}},
		{name: "slash submit", ev: InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventSubmit, Text: "/busy queue"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch := newFakeChannel("telegram")
			m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, &fakeKernel{}, slog.Default())
			if err := m.Register(ch); err != nil {
				t.Fatalf("Register: %v", err)
			}

			if err := m.handleInbound(context.Background(), tc.ev); err != nil {
				t.Fatalf("handleInbound: %v", err)
			}

			sent := ch.sentSnapshot()
			if len(sent) != 1 {
				t.Fatalf("sent messages = %d, want 1: %+v", len(sent), sent)
			}
			if !strings.Contains(sent[0].Text, "Busy input mode set to **queue**") {
				t.Fatalf("busy reply = %q, want queue confirmation", sent[0].Text)
			}
		})
	}
}
