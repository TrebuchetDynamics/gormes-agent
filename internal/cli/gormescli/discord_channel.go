package gormescli

import (
	"context"
	"log/slog"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func CheckDiscordSession(token string) error {
	_, err := discord.NewRealSession(token)
	return err
}

func NewDiscordGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	ds, err := discord.NewRealSession(cfg.Discord.Token)
	if err != nil {
		return nil, err
	}
	return discord.New(discord.Config{
		AllowedChannelID:       cfg.Discord.AllowedChannelID,
		AllowedChannelIDs:      cfg.Discord.AllowedChannelIDs(),
		IgnoredChannelIDs:      cfg.Discord.IgnoredChannelIDs(),
		FreeResponseChannelIDs: cfg.Discord.FreeResponseChannelIDs(),
		NoThreadChannelIDs:     cfg.Discord.NoThreadChannelIDs(),
		ChannelSkillBindings:   cfg.Discord.ChannelSkillBindings,
		ChannelPrompts:         cfg.Discord.ChannelPrompts,
		RequireMention:         cfg.Discord.RequireMentionValue(true),
		RequireMentionSet:      true,
		AutoThread:             cfg.Discord.AutoThreadValue(true),
		AutoThreadSet:          true,
		AllowBots:              cfg.Discord.AllowBotsValue(),
		ReplyToMode:            cfg.Discord.ReplyToModeValue(),
		FirstRunDiscovery:      cfg.Discord.FirstRunDiscovery,
		SkillCollector: func(ctx context.Context) ([]gateway.PlatformCommand, error) {
			return discordGatewaySkillCommands(ctx, cfg)
		},
	}, ds, log), nil
}

func discordGatewaySkillCommands(ctx context.Context, cfg config.Config) ([]gateway.PlatformCommand, error) {
	runtime := skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	skillCommands, _, err := runtime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil || len(skillCommands) == 0 {
		return nil, err
	}
	commands := make([]gateway.PlatformCommand, 0, len(skillCommands))
	for _, cmd := range skillCommands {
		commands = append(commands, gateway.PlatformCommand{
			Name:        strings.TrimPrefix(cmd.Command, "/"),
			Description: cmd.Description,
		})
	}
	return commands, nil
}
