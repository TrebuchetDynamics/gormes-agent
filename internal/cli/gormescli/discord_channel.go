package gormescli

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
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
	}, ds, log), nil
}
