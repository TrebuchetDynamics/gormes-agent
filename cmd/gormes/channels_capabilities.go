package main

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli/modules/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func newChannelsCommand() *cobra.Command {
	return channelsmodule.NewCommandWithSeams(channelsCommandSeams(), channelsCommandOptions())
}

func channelsCommandSeams() channelsmodule.Seams {
	return channelsmodule.Seams{
		LoadConfig:        func() (config.Config, error) { return config.Load(nil) },
		ConfiguredDetails: configuredChannelCapabilityDetails,
	}
}

func channelsCommandOptions() channelsmodule.Options {
	return channelsmodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}

func configuredChannelCapabilityDetails(cfg config.Config) map[string]string {
	details := map[string]string{}
	if cfg.Telegram.BotToken != "" {
		details["telegram"] = configuredTelegramGatewayStatusDetail(cfg.Telegram)
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		details["discord"] = detail
	}
	if cfg.Slack.Enabled {
		details["slack"] = configuredSlackGatewayStatusDetail(cfg.Slack)
	}
	if cfg.Teams.Enabled {
		details["teams"] = configuredTeamsGatewayStatusDetail(cfg.Teams)
	}
	if cfg.Yuanbao.Enabled {
		details["yuanbao"] = cfg.Yuanbao.RedactedStatus()
	}
	return details
}
