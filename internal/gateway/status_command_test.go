package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

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
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-123"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
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
	wantActivity := "Last Activity: " + time.Unix(now.Unix(), 0).Format("2006-01-02 15:04")
	for _, want := range []string{
		"📊 Gormes Gateway Status",
		"Session ID:\nsess-123",
		"Title: (untitled)",
		"Created: (unknown)",
		wantActivity,
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

func TestManagerStatusCommandInitializesMissingChatSession(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		Kind:     EventStatus,
	}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	wantActivity := "Last Activity: " + time.Unix(now.Unix(), 0).Format("2006-01-02 15:04")
	for _, want := range []string{
		"Session ID:\n20260429_094200_",
		wantActivity,
		"Connected Platforms: telegram",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("status response missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Session ID:\n(none)") || strings.Contains(got, "Session ID:\ntelegram:42") {
		t.Fatalf("status response returned invalid session id:\n%s", got)
	}
	mapped, err := smap.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get session map: %v", err)
	}
	if !strings.HasPrefix(mapped, "20260429_094200_") {
		t.Fatalf("session map = %q, want generated Hermes-style session id", mapped)
	}
	meta, ok, err := smap.GetMetadata(ctx, mapped)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("status did not create session metadata")
	}
	if meta.Source != "telegram" || meta.ChatID != "42" || meta.UserID != "user-juan" || meta.UpdatedAt != now.Unix() {
		t.Fatalf("metadata = %+v, want telegram/42/user-juan updated_at=%d", meta, now.Unix())
	}
	if submits := k.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/status submitted to kernel: %#v", submits)
	}
}
