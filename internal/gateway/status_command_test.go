package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestParseInboundTextStatus(t *testing.T) {
	kind, body := ParseInboundText("/status")
	if kind != EventStatus || body != "" {
		t.Fatalf("ParseInboundText(/status) = (%v, %q), want EventStatus empty body", kind, body)
	}
	cmd, ok := ResolveCommand("/status")
	if !ok {
		t.Fatal("/status did not resolve through gateway CommandRegistry")
	}
	if cmd.Kind != EventStatus || cmd.ActiveTurnPolicy != CommandActiveTurnPolicyImmediate {
		t.Fatalf("/status command = (%v, %q), want EventStatus immediate", cmd.Kind, cmd.ActiveTurnPolicy)
	}
}

func TestManagerStatusCommandRendersHermesStyleGatewayStatus(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "telegram:42", "sess-123"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}
	m.rememberUsageFrame(kernel.RenderFrame{
		SessionID: "sess-123",
		Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4},
	})

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	for _, want := range []string{
		"📊 Gormes Gateway Status",
		"Session ID:\nsess-123",
		"Title: (untitled)",
		"Created: (unknown)",
		"Last Activity: (unknown)",
		"Tokens: 7",
		"Agent Running: No",
		"Connected Platforms: telegram",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("status response missing %q in:\n%s", want, got)
		}
	}
	if submits := k.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/status submitted to kernel: %#v", submits)
	}
}
