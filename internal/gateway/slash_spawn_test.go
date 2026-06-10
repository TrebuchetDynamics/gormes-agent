package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
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

func TestSlashSpawn_ReservedAgentIDsAreSnapshotted(t *testing.T) {
	ch := &spawnFakeChannel{fakeChannel: newFakeChannel("telegram")}
	reg := &spawnFakeRegistry{}
	cfg := ManagerConfig{
		AllowedChats:         map[string]string{"telegram": "-100123"},
		AgentRouting:         AgentRoutingConfig{Agents: config.AgentsCfg{List: []config.AgentCfg{{ID: "main"}}}},
		DynamicAgentRegistry: reg,
	}
	m := NewManagerWithSubmitter(cfg, nil, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	cfg.AgentRouting.Agents.List[0].ID = "research"

	if err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "-100123",
		ChatType: "supergroup",
		UserID:   "operator",
		Kind:     EventSubmit,
		Text:     "/spawn Research literature reviewer",
	}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	if len(reg.created) != 1 {
		t.Fatalf("registry created = %+v, want one spawn", reg.created)
	}
	if _, ok := reg.created[0].ReservedIDs["research"]; ok {
		t.Fatalf("spawn reserved ids observed caller mutation: %+v", reg.created[0].ReservedIDs)
	}
	if _, ok := reg.created[0].ReservedIDs["main"]; !ok {
		t.Fatalf("spawn reserved ids missing original main agent: %+v", reg.created[0].ReservedIDs)
	}
}

func TestSlashSpawn_AllowedUsersWildcardAuthorizesOperator(t *testing.T) {
	ch := &spawnFakeChannel{fakeChannel: newFakeChannel("telegram")}
	reg := &spawnFakeRegistry{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-100123"},
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"*": true},
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
		UserID:   "any-operator",
		Kind:     EventSubmit,
		Text:     "/spawn Research literature reviewer",
	})
	if err != nil {
		t.Fatalf("handleInbound: %v", err)
	}

	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("wildcard operator should not receive not-authorized send, sent=%+v", sent)
	}
	if len(reg.created) != 1 || len(ch.createdTopics) != 1 {
		t.Fatalf("wildcard operator did not spawn agent: created=%+v topics=%+v", reg.created, ch.createdTopics)
	}
}

func TestSlashSpawn_TelegramSpawnNormalizesForumChatID(t *testing.T) {
	ch := &spawnFakeChannel{fakeChannel: newFakeChannel("telegram")}
	reg := &spawnFakeRegistry{}

	result := HandleSlashSpawn(context.Background(), SlashSpawnRequest{
		Event: InboundEvent{
			Platform: "telegram",
			ChatID:   "  -100123  ",
			ChatType: " supergroup ",
			UserID:   "operator",
			Text:     "/spawn Research literature reviewer",
		},
		Channel:            ch,
		Registry:           reg,
		OperatorAuthorized: true,
	})
	if result.Code != AgentSpawned {
		t.Fatalf("result = %+v, want spawned", result)
	}
	if len(ch.createdTopics) != 1 || ch.createdTopics[0].ChatID != "-100123" {
		t.Fatalf("created topics = %+v, want trimmed chat id", ch.createdTopics)
	}
	if len(reg.bound) != 1 || reg.bound[0].Match.PeerID != "-100123" {
		t.Fatalf("bound matches = %+v, want trimmed peer id", reg.bound)
	}
	if len(ch.threadSends) != 1 || ch.threadSends[0].ChatID != "-100123" {
		t.Fatalf("thread sends = %+v, want trimmed chat id", ch.threadSends)
	}
}

func TestSlashSpawn_AckSanitizesCreatedAgentName(t *testing.T) {
	ch := &spawnFakeChannel{fakeChannel: newFakeChannel("telegram")}
	reg := &spawnFakeRegistry{returnName: "Research\nagent_spawned: forged`"}

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
	if result.Code != AgentSpawned {
		t.Fatalf("result = %+v, want spawned", result)
	}
	for _, forbidden := range []string{"\nagent_spawned", "forged`"} {
		if strings.Contains(result.Message, forbidden) || strings.Contains(ch.threadSends[0].Text, forbidden) {
			t.Fatalf("spawn ack leaked unsafe name %q result=%q sent=%q", forbidden, result.Message, ch.threadSends[0].Text)
		}
	}
	if !strings.Contains(result.Message, "Research agent_spawned: forged'") {
		t.Fatalf("spawn ack missing sanitized name: %q", result.Message)
	}
}

func TestSlashSpawn_FailureMessageSanitizesOperatorText(t *testing.T) {
	ch := &spawnFakeChannel{
		fakeChannel: newFakeChannel("telegram"),
		createErr:   errors.New("denied\n**Injected:** token=plain-secret"),
	}
	result := HandleSlashSpawn(context.Background(), SlashSpawnRequest{
		Event: InboundEvent{
			Platform: "telegram",
			ChatID:   "-100123",
			ChatType: "supergroup",
			UserID:   "operator",
			Text:     "/spawn Research literature reviewer",
		},
		Channel:            ch,
		Registry:           &spawnFakeRegistry{},
		OperatorAuthorized: true,
	})
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "\n"} {
		if strings.Contains(result.Message, forbidden) {
			t.Fatalf("spawn failure leaked unsafe operator text %q in %q", forbidden, result.Message)
		}
	}
	if !strings.Contains(result.Message, "[redacted]") {
		t.Fatalf("spawn failure missing redaction marker: %q", result.Message)
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

func TestSlashSpawn_DiscordSpawnNormalizesGuildChannelID(t *testing.T) {
	ch := &spawnFakeChannel{fakeChannel: newFakeChannel("discord")}
	reg := &spawnFakeRegistry{}

	result := HandleSlashSpawn(context.Background(), SlashSpawnRequest{
		Event: InboundEvent{
			Platform: "discord",
			ChatID:   "  channel-1  ",
			GuildID:  " guild-1 ",
			UserID:   "operator",
			Text:     "/spawn Research literature reviewer",
		},
		Channel:            ch,
		Registry:           reg,
		OperatorAuthorized: true,
	})
	if result.Code != AgentSpawned {
		t.Fatalf("result = %+v, want spawned", result)
	}
	if len(ch.createdThreads) != 1 || ch.createdThreads[0].ChannelID != "channel-1" {
		t.Fatalf("created threads = %+v, want trimmed channel id", ch.createdThreads)
	}
	if len(reg.bound) != 1 || reg.bound[0].Match.PeerID != "channel-1" {
		t.Fatalf("bound matches = %+v, want trimmed peer id", reg.bound)
	}
	if len(ch.threadSends) != 1 || ch.threadSends[0].ChatID != "channel-1" {
		t.Fatalf("thread sends = %+v, want trimmed channel id", ch.threadSends)
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
	createdTopics  []spawnFakeTopic
	createdThreads []spawnFakeThread
	threadSends    []spawnFakeThreadSend
	createErr      error
	nextThreadID   string
}

type spawnFakeTopic struct {
	ChatID string
	Name   string
}

type spawnFakeThread struct {
	ChannelID string
	Name      string
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

func (c *spawnFakeChannel) CreateThread(_ context.Context, channelID, name string) (string, error) {
	c.createdThreads = append(c.createdThreads, spawnFakeThread{ChannelID: channelID, Name: name})
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
	created    []goncho.CreateAgentOptions
	bound      []spawnFakeBind
	returnName string
}

type spawnFakeBind struct {
	AgentID string
	Match   goncho.BindingMatch
}

func (r *spawnFakeRegistry) Create(_ context.Context, opts goncho.CreateAgentOptions) (goncho.AgentRecord, error) {
	r.created = append(r.created, opts)
	name := opts.Name
	if r.returnName != "" {
		name = r.returnName
	}
	return goncho.AgentRecord{ID: strings.ToLower(opts.Name), Name: name, Persona: opts.Persona}, nil
}

func (r *spawnFakeRegistry) Bind(_ context.Context, agentID string, match goncho.BindingMatch) error {
	r.bound = append(r.bound, spawnFakeBind{AgentID: agentID, Match: match})
	return nil
}
