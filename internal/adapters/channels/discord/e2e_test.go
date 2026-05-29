package discord

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestDiscordGatewayE2EAccountChannelRoutesAndDeliversFinal(t *testing.T) {
	ms := newMockSession()
	bot := New(Config{AllowedChannelID: "dm-ops", AccountID: "ops"}, ms, nil)
	if got := bot.Name(); got != "discord:ops" {
		t.Fatalf("bot name = %q, want account-scoped platform", got)
	}

	provider := llm.NewMockClient()
	provider.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Discord account final"},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "sess-discord-e2e")
	k := kernel.New(kernel.Config{
		Model:     "mock-discord-model",
		Endpoint:  "http://mock-provider",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), slog.Default())

	mgr := gateway.NewManager(gateway.ManagerConfig{
		AllowedChats: map[string]string{"discord:ops": "dm-ops"},
		CoalesceMs:   5,
	}, k, slog.Default())
	if err := mgr.Register(bot); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	go func() { _ = mgr.Run(ctx) }()
	ms.waitOpen(t)

	if !ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "msg-1",
		ChannelID: "dm-ops",
		Content:   "hello from the ops discord account",
		Author:    &discordgo.User{ID: "user-1", Username: "operator"},
	}}) {
		t.Fatal("mock Discord session did not find a MessageCreate handler")
	}

	waitForDiscordE2E(t, 3*time.Second, func() bool { return len(provider.Requests()) == 1 })
	req := provider.Requests()[0]
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" || last.Content != "hello from the ops discord account" {
		t.Fatalf("provider final request message = %+v, want Discord user text", last)
	}

	waitForDiscordE2E(t, 3*time.Second, func() bool {
		return strings.Contains(lastDiscordVisibleText(ms.complexSnapshot(), ms.editsSnapshot()), "Discord account final")
	})
}

func TestDiscordInboundEventsUseAccountScopedPlatform(t *testing.T) {
	b := New(Config{AllowedChannelID: "dm-ops", AccountID: "ops"}, newMockSession(), nil)
	ev, ok := b.toInboundEvent(&discordgo.Message{
		ID:        "msg-1",
		ChannelID: "dm-ops",
		Content:   "hello",
		Author:    &discordgo.User{ID: "user-1"},
	})
	if !ok {
		t.Fatal("toInboundEvent rejected account message")
	}
	if ev.Platform != "discord:ops" {
		t.Fatalf("Platform = %q, want account-scoped channel name", ev.Platform)
	}
	if ev.AccountID != "ops" {
		t.Fatalf("AccountID = %q, want ops", ev.AccountID)
	}

	thread, ok := b.toThreadLifecycleEvent(&discordgo.Channel{
		ID:       "thread-1",
		ParentID: "parent-1",
		Name:     "agent thread",
		Type:     discordgo.ChannelTypeGuildPublicThread,
	})
	if !ok {
		t.Fatal("toThreadLifecycleEvent rejected account thread")
	}
	if thread.Platform != "discord:ops" || thread.AccountID != "ops" {
		t.Fatalf("thread event Platform/AccountID = %q/%q, want discord:ops/ops", thread.Platform, thread.AccountID)
	}
}

func lastDiscordVisibleText(sent []mockComplexSent, edits []mockEdit) string {
	for i := len(edits) - 1; i >= 0; i-- {
		if strings.TrimSpace(edits[i].Content) != "" {
			return edits[i].Content
		}
	}
	for i := len(sent) - 1; i >= 0; i-- {
		if sent[i].Data != nil && strings.TrimSpace(sent[i].Data.Content) != "" {
			return sent[i].Data.Content
		}
	}
	return ""
}

func waitForDiscordE2E(t *testing.T, timeout time.Duration, ok func() bool) {
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
