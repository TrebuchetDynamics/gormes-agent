package gateway

import (
	"context"
	"strings"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	gatewayspawn "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/spawncmd"
)

type AgentSpawnEvidence string

const (
	AgentSpawned                   AgentSpawnEvidence = "agent_spawned"
	AgentSpawnInvalid              AgentSpawnEvidence = "agent_spawn_invalid"
	AgentSpawnNotAuthorized        AgentSpawnEvidence = "agent_spawn_not_authorized"
	AgentSpawnRequiresForum        AgentSpawnEvidence = "agent_spawn_requires_forum"
	AgentSpawnRequiresGuildChannel AgentSpawnEvidence = "agent_spawn_requires_guild_channel"
	AgentSpawnUnavailable          AgentSpawnEvidence = "agent_spawn_unavailable"
	AgentSpawnTopicFailed          AgentSpawnEvidence = "agent_spawn_topic_failed"
	AgentSpawnThreadFailed         AgentSpawnEvidence = "agent_spawn_thread_failed"
	AgentSpawnRegistryFailed       AgentSpawnEvidence = "agent_spawn_registry_failed"
	AgentSpawnAckFailed            AgentSpawnEvidence = "agent_spawn_ack_failed"
)

var errSpawnUsage = gatewayspawn.ErrUsage

// SpawnAgentRegistry is the small part of Goncho's dynamic registry needed by
// channel-native /spawn flows.
type SpawnAgentRegistry interface {
	Create(context.Context, goncho.CreateAgentOptions) (goncho.AgentRecord, error)
	Bind(context.Context, string, goncho.BindingMatch) error
}

// TelegramForumTopicCreator is implemented by Telegram channels that can call
// createForumTopic through the Bot API.
type TelegramForumTopicCreator interface {
	CreateForumTopic(ctx context.Context, chatID, name string) (threadID string, err error)
}

// DiscordThreadCreator is implemented by Discord channels that can create a
// new public thread under a guild text channel.
type DiscordThreadCreator interface {
	CreateThread(ctx context.Context, channelID, name string) (threadID string, err error)
}

type SpawnSlashCommand = gatewayspawn.Command

type SlashSpawnRequest struct {
	Event              InboundEvent
	Channel            Channel
	Registry           SpawnAgentRegistry
	OperatorAuthorized bool
	ReservedIDs        map[string]struct{}
}

type SlashSpawnResult struct {
	Code      AgentSpawnEvidence
	Message   string
	AgentID   string
	ThreadID  string
	Delivered bool
}

func ParseSpawnSlash(raw string) (SpawnSlashCommand, error) {
	return gatewayspawn.Parse(raw)
}

func HandleSlashSpawn(ctx context.Context, req SlashSpawnRequest) SlashSpawnResult {
	if !req.OperatorAuthorized {
		return spawnResult(AgentSpawnNotAuthorized, string(AgentSpawnNotAuthorized)+": /spawn is restricted to operator-tier users.")
	}
	cmd, err := ParseSpawnSlash(req.Event.Text)
	if err != nil {
		return spawnResult(AgentSpawnInvalid, string(AgentSpawnInvalid)+": "+err.Error())
	}
	if req.Registry == nil {
		return spawnResult(AgentSpawnUnavailable, string(AgentSpawnUnavailable)+": dynamic agent registry is not configured.")
	}

	if isTelegramPlatform(req.Event.Platform) {
		return handleTelegramSlashSpawn(ctx, req, cmd)
	}
	if isDiscordPlatform(req.Event.Platform) {
		return handleDiscordSlashSpawn(ctx, req, cmd)
	}
	return spawnResult(AgentSpawnUnavailable, string(AgentSpawnUnavailable)+": /spawn is not available for this platform.")
}

func handleTelegramSlashSpawn(ctx context.Context, req SlashSpawnRequest, cmd SpawnSlashCommand) SlashSpawnResult {
	chatID, ok := telegramSpawnForumChatID(req.Event)
	if !ok {
		return spawnResult(AgentSpawnRequiresForum, string(AgentSpawnRequiresForum)+": /spawn requires a Telegram forum-enabled supergroup.")
	}
	topicCreator, ok := req.Channel.(TelegramForumTopicCreator)
	if !ok {
		return spawnResult(AgentSpawnRequiresForum, string(AgentSpawnRequiresForum)+": Telegram forum topic creation is unavailable for this channel.")
	}
	threadSender, ok := req.Channel.(ThreadSender)
	if !ok {
		return spawnResult(AgentSpawnUnavailable, string(AgentSpawnUnavailable)+": Telegram thread sends are unavailable for this channel.")
	}

	threadID, err := topicCreator.CreateForumTopic(ctx, chatID, cmd.Name)
	if err != nil {
		return spawnResult(AgentSpawnTopicFailed, string(AgentSpawnTopicFailed)+": createForumTopic failed: "+spawnErrorText(err))
	}
	created, err := req.Registry.Create(ctx, goncho.CreateAgentOptions{
		Name:        cmd.Name,
		Persona:     cmd.Persona,
		ReservedIDs: req.ReservedIDs,
	})
	if err != nil {
		return spawnResult(AgentSpawnRegistryFailed, string(AgentSpawnRegistryFailed)+": "+spawnErrorText(err))
	}
	match := goncho.BindingMatch{
		Channel:  "telegram",
		PeerKind: "group",
		PeerID:   chatID,
		ThreadID: threadID,
	}
	if err := req.Registry.Bind(ctx, created.ID, match); err != nil {
		return spawnResult(AgentSpawnRegistryFailed, string(AgentSpawnRegistryFailed)+": "+spawnErrorText(err))
	}

	ack := string(AgentSpawned) + ": " + spawnLineValue(created.Name) + " is ready in this topic."
	if _, err := threadSender.SendThread(ctx, chatID, threadID, ack); err != nil {
		return spawnResult(AgentSpawnAckFailed, string(AgentSpawnAckFailed)+": "+spawnErrorText(err))
	}
	return SlashSpawnResult{
		Code:      AgentSpawned,
		Message:   ack,
		AgentID:   created.ID,
		ThreadID:  threadID,
		Delivered: true,
	}
}

func telegramSpawnForumSurface(ev InboundEvent) bool {
	_, ok := telegramSpawnForumChatID(ev)
	return ok
}

func telegramSpawnForumChatID(ev InboundEvent) (string, bool) {
	if !isTelegramPlatform(ev.Platform) {
		return "", false
	}
	chatID := strings.TrimSpace(ev.ChatID)
	return chatID, strings.EqualFold(strings.TrimSpace(ev.ChatType), "supergroup") && chatID != ""
}

func handleDiscordSlashSpawn(ctx context.Context, req SlashSpawnRequest, cmd SpawnSlashCommand) SlashSpawnResult {
	channelID, ok := discordSpawnGuildChannelID(req.Event)
	if !ok {
		return spawnResult(AgentSpawnRequiresGuildChannel, string(AgentSpawnRequiresGuildChannel)+": /spawn requires a Discord guild text channel.")
	}
	threadCreator, ok := req.Channel.(DiscordThreadCreator)
	if !ok {
		return spawnResult(AgentSpawnUnavailable, string(AgentSpawnUnavailable)+": Discord thread creation is unavailable for this channel.")
	}
	threadSender, ok := req.Channel.(ThreadSender)
	if !ok {
		return spawnResult(AgentSpawnUnavailable, string(AgentSpawnUnavailable)+": Discord thread sends are unavailable for this channel.")
	}

	threadID, err := threadCreator.CreateThread(ctx, channelID, cmd.Name)
	if err != nil {
		return spawnResult(AgentSpawnThreadFailed, string(AgentSpawnThreadFailed)+": StartThread failed: "+spawnErrorText(err))
	}
	created, err := req.Registry.Create(ctx, goncho.CreateAgentOptions{
		Name:        cmd.Name,
		Persona:     cmd.Persona,
		ReservedIDs: req.ReservedIDs,
	})
	if err != nil {
		return spawnResult(AgentSpawnRegistryFailed, string(AgentSpawnRegistryFailed)+": "+spawnErrorText(err))
	}
	match := goncho.BindingMatch{
		Channel:  "discord",
		PeerKind: "channel",
		PeerID:   channelID,
		ThreadID: threadID,
	}
	if err := req.Registry.Bind(ctx, created.ID, match); err != nil {
		return spawnResult(AgentSpawnRegistryFailed, string(AgentSpawnRegistryFailed)+": "+spawnErrorText(err))
	}

	ack := string(AgentSpawned) + ": " + spawnLineValue(created.Name) + " is ready in this thread."
	if _, err := threadSender.SendThread(ctx, channelID, threadID, ack); err != nil {
		return spawnResult(AgentSpawnAckFailed, string(AgentSpawnAckFailed)+": "+spawnErrorText(err))
	}
	return SlashSpawnResult{
		Code:      AgentSpawned,
		Message:   ack,
		AgentID:   created.ID,
		ThreadID:  threadID,
		Delivered: true,
	}
}

func discordSpawnGuildSurface(ev InboundEvent) bool {
	_, ok := discordSpawnGuildChannelID(ev)
	return ok
}

func discordSpawnGuildChannelID(ev InboundEvent) (string, bool) {
	if !isDiscordPlatform(ev.Platform) {
		return "", false
	}
	channelID := strings.TrimSpace(ev.ChatID)
	return channelID, strings.TrimSpace(ev.GuildID) != "" && channelID != ""
}

func spawnLineValue(value string) string {
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func spawnErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactSpawnSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(compact, marker) || strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	return spawnLineValue(msg)
}

func compactSpawnSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func spawnResult(code AgentSpawnEvidence, msg string) SlashSpawnResult {
	return SlashSpawnResult{Code: code, Message: msg}
}

func (m *Manager) handleSpawnCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	result := HandleSlashSpawn(ctx, SlashSpawnRequest{
		Event:              ev,
		Channel:            ch,
		Registry:           m.cfg.DynamicAgentRegistry,
		OperatorAuthorized: m.spawnOperatorAuthorized(ev),
		ReservedIDs:        m.spawnReservedAgentIDs(),
	})
	if !result.Delivered && strings.TrimSpace(result.Message) != "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, result.Message)
	}
}

func (m *Manager) spawnOperatorAuthorized(ev InboundEvent) bool {
	if users := m.cfg.AllowedUsers[ev.Platform]; len(users) > 0 {
		if users["*"] {
			return true
		}
		return users[strings.TrimSpace(ev.UserID)]
	}
	return true
}

func (m *Manager) spawnReservedAgentIDs() map[string]struct{} {
	agents := m.cfg.AgentRouting.Agents.List
	if len(agents) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		id := strings.TrimSpace(agent.ID)
		if id != "" {
			out[id] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
