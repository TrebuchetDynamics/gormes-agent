package slack

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestSlackGatewayE2EAccountChannelRoutesAndDeliversFinal(t *testing.T) {
	mc := newMockClient()
	ch := NewChannel(mc, nil, ChannelConfig{AccountID: "team-a", RequireMention: false})
	if got := ch.Name(); got != "slack:team-a" {
		t.Fatalf("channel name = %q, want account-scoped platform", got)
	}

	provider := hermes.NewMockClient()
	provider.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "Slack account final"},
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-slack-e2e")
	k := kernel.New(kernel.Config{
		Model:     "mock-slack-model",
		Endpoint:  "http://mock-provider",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), slog.Default())

	mgr := gateway.NewManager(gateway.ManagerConfig{
		AllowedChats: map[string]string{"slack:team-a": "C-team-a"},
		CoalesceMs:   5,
	}, k, slog.Default())
	if err := mgr.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	go func() { _ = mgr.Run(ctx) }()

	mc.pushEvent(Event{
		RequestID: "req-1",
		ChannelID: "C-team-a",
		TeamID:    "T-from-event",
		UserID:    "U-user",
		Text:      "hello from the team-a slack account",
		ChatType:  "im",
		Timestamp: "1711111111.000001",
	})

	waitForSlackE2E(t, time.Second, func() bool { return len(provider.Requests()) == 1 })
	req := provider.Requests()[0]
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" || last.Content != "hello from the team-a slack account" {
		t.Fatalf("provider final request message = %+v, want Slack user text", last)
	}

	waitForSlackE2E(t, time.Second, func() bool {
		return strings.Contains(mc.lastOutputText(), "Slack account final")
	})
}

func TestSlackInboundEventsUseAccountScopedPlatform(t *testing.T) {
	ch := NewChannel(newMockClient(), nil, ChannelConfig{AccountID: "team-a", RequireMention: false})
	ch.selfUserID = "UBOT"
	ev, ok := ch.toInboundEvent(Event{
		RequestID: "req-1",
		ChannelID: "C-team-a",
		TeamID:    "T-from-event",
		UserID:    "U-user",
		Text:      "hello",
		ChatType:  "im",
		Timestamp: "1711111111.000001",
	})
	if !ok {
		t.Fatal("toInboundEvent rejected account message")
	}
	if ev.Platform != "slack:team-a" {
		t.Fatalf("Platform = %q, want account-scoped channel name", ev.Platform)
	}
	if ev.AccountID != "team-a" {
		t.Fatalf("AccountID = %q, want team-a", ev.AccountID)
	}
}

func waitForSlackE2E(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
