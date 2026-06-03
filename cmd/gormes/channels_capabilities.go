package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
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
	return gormescli.ConfiguredChannelCapabilityDetails(cfg)
}

func configuredWhatsAppGatewayStatusDetail() string {
	return gormescli.ConfiguredWhatsAppGatewayStatusDetail()
}
