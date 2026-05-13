package discord

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestDiscordSpawnSlash_CreatesThreadAndBindsAgent(t *testing.T) {
	ctx := context.Background()
	ms := newMockSession()
	ms.threadStartResult = &discordgo.Channel{
		ID:       "thread-777",
		ParentID: "chan-1",
		Name:     "Research",
		GuildID:  "guild-1",
		Type:     discordgo.ChannelTypeGuildPublicThread,
	}
	reg := newDiscordSpawnRegistry(t)
	bot := New(Config{AllowedChannelID: "chan-1", RequireMention: false, RequireMentionSet: true}, ms, nil)
	cancel, done := runDiscordSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"discord": "chan-1"},
		AllowedUsers: map[string]map[string]bool{
			"discord": {"operator": true},
		},
	})
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordSpawnMessage("chan-1", "guild-1", "operator", "/spawn Research literature reviewer"))

	waitForDiscordSpawn(t, func() bool {
		agentID, found, err := reg.Resolve(ctx, goncho.BindingMatch{
			Channel:  "discord",
			PeerKind: "channel",
			PeerID:   "chan-1",
			ThreadID: "thread-777",
		})
		return err == nil && found && agentID == "research" && len(ms.complexSnapshot()) >= 1
	})

	starts := ms.threadStartCallsSnapshot()
	if len(starts) != 1 {
		t.Fatalf("thread starts = %+v, want one direct StartThread call", starts)
	}
	if starts[0].ChannelID != "chan-1" || starts[0].Data == nil || starts[0].Data.Name != "Research" {
		t.Fatalf("thread start = %+v, want channel/name", starts[0])
	}
	if len(ms.messageThreadStartCallsSnapshot()) != 0 {
		t.Fatalf("message thread fallback used unexpectedly: %+v", ms.messageThreadStartCallsSnapshot())
	}
	sent := ms.complexSnapshot()
	if sent[0].ChannelID != "thread-777" || sent[0].Data == nil || !strings.Contains(sent[0].Data.Content, "agent_spawned") {
		t.Fatalf("ack send = %+v, want agent_spawned inside thread", sent[0])
	}
	rec, found, err := reg.Get(ctx, "research")
	if err != nil || !found {
		t.Fatalf("Get(research) = %+v, %v, %v; want record", rec, found, err)
	}
	if rec.Persona != "literature reviewer" {
		t.Fatalf("persona = %q, want literature reviewer", rec.Persona)
	}
}

func TestDiscordSpawnSlash_RejectsDirectMessage(t *testing.T) {
	ms := newMockSession()
	reg := newDiscordSpawnRegistry(t)
	bot := New(Config{AllowedChannelID: "dm-1", RequireMention: false, RequireMentionSet: true}, ms, nil)
	cancel, done := runDiscordSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"discord": "dm-1"},
		AllowedUsers: map[string]map[string]bool{
			"discord": {"operator": true},
		},
	})
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordSpawnMessage("dm-1", "", "operator", "/spawn Research literature reviewer"))

	waitForDiscordSpawn(t, func() bool {
		return discordSentTextContains(ms, "agent_spawn_requires_guild_channel")
	})
	if len(ms.threadStartCallsSnapshot()) != 0 {
		t.Fatalf("thread starts = %+v, want none for DM", ms.threadStartCallsSnapshot())
	}
	assertDiscordSpawnRegistryEmpty(t, reg)
}

func TestDiscordSpawnSlash_RollsBackOnThreadCreationFailure(t *testing.T) {
	ms := newMockSession()
	ms.threadStartErr = errors.New("missing permissions")
	reg := newDiscordSpawnRegistry(t)
	bot := New(Config{AllowedChannelID: "chan-1", RequireMention: false, RequireMentionSet: true}, ms, nil)
	cancel, done := runDiscordSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"discord": "chan-1"},
		AllowedUsers: map[string]map[string]bool{
			"discord": {"operator": true},
		},
	})
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordSpawnMessage("chan-1", "guild-1", "operator", "/spawn Research literature reviewer"))

	waitForDiscordSpawn(t, func() bool {
		return discordSentTextContains(ms, "agent_spawn_thread_failed") && discordSentTextContains(ms, "missing permissions")
	})
	if len(ms.threadStartCallsSnapshot()) != 1 {
		t.Fatalf("thread starts = %+v, want one failing StartThread call", ms.threadStartCallsSnapshot())
	}
	if len(ms.messageThreadStartCallsSnapshot()) != 0 {
		t.Fatalf("message thread fallback used unexpectedly: %+v", ms.messageThreadStartCallsSnapshot())
	}
	assertDiscordSpawnRegistryEmpty(t, reg)
}

func TestDiscordSpawnSlash_RejectsNonOperatorTier(t *testing.T) {
	ms := newMockSession()
	reg := newDiscordSpawnRegistry(t)
	bot := New(Config{AllowedChannelID: "chan-1", RequireMention: false, RequireMentionSet: true}, ms, nil)
	cancel, done := runDiscordSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"discord": "chan-1"},
		AllowedUsers: map[string]map[string]bool{
			"discord": {"operator": true},
		},
	})
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordSpawnMessage("chan-1", "guild-1", "stranger", "/spawn Research literature reviewer"))

	waitForDiscordSpawn(t, func() bool {
		return discordSentTextContains(ms, "agent_spawn_not_authorized")
	})
	if len(ms.threadStartCallsSnapshot()) != 0 {
		t.Fatalf("thread starts = %+v, want none for non-operator", ms.threadStartCallsSnapshot())
	}
	assertDiscordSpawnRegistryEmpty(t, reg)
}

func newDiscordSpawnRegistry(t *testing.T) *goncho.DynamicAgentRegistry {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	reg, err := goncho.NewDynamicAgentRegistry(store.DB())
	if err != nil {
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	return reg
}

func runDiscordSpawnManager(t *testing.T, bot *Bot, reg *goncho.DynamicAgentRegistry, cfg gateway.ManagerConfig) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	cfg.DynamicAgentRegistry = reg
	m := gateway.NewManagerWithSubmitter(cfg, nil, slog.Default())
	if err := m.Register(bot); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = m.Run(ctx)
		close(done)
	}()
	bot.session.(*mockSession).waitOpen(t)
	return cancel, done
}

func discordSpawnMessage(channelID, guildID, userID, text string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "msg-1",
		ChannelID: channelID,
		GuildID:   guildID,
		Content:   text,
		Author:    &discordgo.User{ID: userID},
		Type:      discordgo.MessageTypeDefault,
	}}
}

func waitForDiscordSpawn(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func discordSentTextContains(ms *mockSession, needle string) bool {
	for _, sent := range ms.complexSnapshot() {
		if sent.Data != nil && strings.Contains(sent.Data.Content, needle) {
			return true
		}
	}
	for _, sent := range ms.sentSnapshot() {
		if strings.Contains(sent.Content, needle) {
			return true
		}
	}
	return false
}

func assertDiscordSpawnRegistryEmpty(t *testing.T, reg *goncho.DynamicAgentRegistry) {
	t.Helper()
	records, err := reg.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("registry records = %+v, want empty", records)
	}
}
