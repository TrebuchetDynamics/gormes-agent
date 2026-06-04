package main

import (
	"github.com/spf13/cobra"

	channelscmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

func newChannelsCommand() *cobra.Command {
	return channelsmodule.NewCommandWithSeams(channelsCommandSeams(), channelsCommandOptions())
}

func channelsCommandSeams() channelsmodule.Seams {
	return channelscmd.Seams()
}

func channelsCommandOptions() channelsmodule.Options {
	return channelscmd.Options(func() channelscmd.BuildProvenance {
		build := newBuildProvenance()
		return channelscmd.BuildProvenance{
			Version:   build.Version,
			GitCommit: build.GitCommit,
		}
	})
}

func configuredChannelCapabilityDetails(cfg config.Config) map[string]string {
	return channelscmd.ConfiguredChannelCapabilityDetails(cfg)
}

func configuredWhatsAppGatewayStatusDetail() string {
	return channelscmd.ConfiguredWhatsAppGatewayStatusDetail()
}
