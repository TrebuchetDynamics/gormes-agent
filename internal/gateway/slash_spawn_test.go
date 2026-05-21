package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho"
)

func TestSlashSpawn_ParseNameAndPersona(t *testing.T) {
	cmd, err := ParseSpawnSlash("/spawn Research literature reviewer")
	if err != nil {
		t.Fatalf("ParseSpawnSlash: %v", err)
	}
	if cmd.Name != "Research" || cmd.Persona != "literature reviewer" {
		t.Fatalf("parsed spawn = %+v, want name Research and persona literature reviewer", cmd)
	}
}

func TestSlashSpawn_CommandRegistryPreservesArgs(t *testing.T) {
	kind, body := ParseInboundText("/spawn Research literature reviewer")
	if kind != EventSpawn || body != "/spawn Research literature reviewer" {
		t.Fatalf("ParseInboundText(/spawn ...) = (%v, %q), want EventSpawn with raw body", kind, body)
	}
}

func TestSlashSpawn_RejectsNonOperatorTier(t *testing.T) {
	ch := &spawnFakeChannel{fakeChannel: newFakeChannel("telegram")}
	reg := &spawnFakeRegistry{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-100123"},
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"operator": true},
		},
		DynamicAgentRegistry: reg,
	}, nil, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "-100123",
		ChatType: "supergroup",
		UserID:   "stranger",
		Kind:     EventSubmit,
		Text:     "/spawn Research literature reviewer",
	})
	if err != nil {
		t.Fatalf("handleInbound: %v", err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "agent_spawn_not_authorized") {
		t.Fatalf("sent = %+v, want agent_spawn_not_authorized evidence", sent)
	}
	if len(reg.created) != 0 || len(ch.createdTopics) != 0 {
		t.Fatalf("non-operator mutated state: created=%+v topics=%+v", reg.created, ch.createdTopics)
	}
}

func TestSlashSpawn_TelegramTopicFailureDoesNotCreateAgent(t *testing.T) {
	ch := &spawnFakeChannel{
		fakeChannel: newFakeChannel("telegram"),
		createErr:   errors.New("Bad Request: not enough rights to manage topics"),
	}
	reg := &spawnFakeRegistry{}

	result := HandleSlashSpawn(context.Background(), SlashSpawnRequest{
		Event: InboundEvent{
			Platform: "telegram",
			ChatID:   "-100123",
			ChatType: "supergroup",
			UserID:   "operator",
			Text:     "/spawn Research literature reviewer",
		},
		Channel:            ch,
		Registry:           reg,
		OperatorAuthorized: true,
	})
	if result.Code != AgentSpawnTopicFailed || !strings.Contains(result.Message, "not enough rights") {
		t.Fatalf("result = %+v, want topic failure with upstream error", result)
	}
	if len(reg.created) != 0 {
		t.Fatalf("registry created = %+v, want empty when topic creation fails", reg.created)
	}
}

func TestSlashSpawn_DiscordRejectsDirectMessage(t *testing.T) {
	result := HandleSlashSpawn(context.Background(), SlashSpawnRequest{
		Event: InboundEvent{
			Platform: "discord",
			ChatID:   "dm-1",
			UserID:   "operator",
			Text:     "/spawn Research literature reviewer",
		},
		Channel:            &spawnFakeChannel{fakeChannel: newFakeChannel("discord")},
		Registry:           &spawnFakeRegistry{},
		OperatorAuthorized: true,
	})
	if result.Code != AgentSpawnRequiresGuildChannel || !strings.Contains(result.Message, "agent_spawn_requires_guild_channel") {
		t.Fatalf("result = %+v, want guild-channel rejection", result)
	}
}

type spawnFakeChannel struct {
	*fakeChannel
	createdTopics []spawnFakeTopic
	threadSends   []spawnFakeThreadSend
	createErr     error
	nextThreadID  string
}

type spawnFakeTopic struct {
	ChatID string
	Name   string
}

type spawnFakeThreadSend struct {
	ChatID   string
	ThreadID string
	Text     string
}

func (c *spawnFakeChannel) CreateForumTopic(_ context.Context, chatID, name string) (string, error) {
	c.createdTopics = append(c.createdTopics, spawnFakeTopic{ChatID: chatID, Name: name})
	if c.createErr != nil {
		return "", c.createErr
	}
	if c.nextThreadID == "" {
		c.nextThreadID = "777"
	}
	return c.nextThreadID, nil
}

func (c *spawnFakeChannel) SendThread(_ context.Context, chatID, threadID, text string) (string, error) {
	c.threadSends = append(c.threadSends, spawnFakeThreadSend{ChatID: chatID, ThreadID: threadID, Text: text})
	return "thread-msg-1", nil
}

type spawnFakeRegistry struct {
	created []goncho.CreateAgentOptions
	bound   []spawnFakeBind
}

type spawnFakeBind struct {
	AgentID string
	Match   goncho.BindingMatch
}

func (r *spawnFakeRegistry) Create(_ context.Context, opts goncho.CreateAgentOptions) (goncho.AgentRecord, error) {
	r.created = append(r.created, opts)
	return goncho.AgentRecord{ID: strings.ToLower(opts.Name), Name: opts.Name, Persona: opts.Persona}, nil
}

func (r *spawnFakeRegistry) Bind(_ context.Context, agentID string, match goncho.BindingMatch) error {
	r.bound = append(r.bound, spawnFakeBind{AgentID: agentID, Match: match})
	return nil
}
