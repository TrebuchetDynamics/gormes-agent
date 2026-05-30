package channelruntime

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/slack"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// NewSlackGatewayChannel binds gateway Slack config to the live Slack channel.
func NewSlackGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	return slack.NewChannel(slack.NewRealClient(cfg.Slack.BotToken, cfg.Slack.AppToken), log, slack.ChannelConfig{
		RequireMention:       cfg.Slack.RequireMention,
		StrictMention:        cfg.Slack.StrictMention,
		FreeResponseChannels: cfg.Slack.FreeResponseChannels,
		ChannelSkillBindings: cfg.Slack.ChannelSkillBindings,
		ChannelPrompts:       cfg.Slack.ChannelPrompts,
		AccountID:            cfg.Slack.AccountID,
	}), nil
}
