package channels

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

type BuildProvenance = gormescli.BuildProvenance

func Seams() channelsmodule.Seams {
	return channelsmodule.Seams{
		LoadConfig:        func() (config.Config, error) { return config.Load(nil) },
		ConfiguredDetails: ConfiguredChannelCapabilityDetails,
	}
}

func Options(build func() BuildProvenance) channelsmodule.Options {
	return channelsmodule.Options{BuildProvenance: build}
}

func ConfiguredChannelCapabilityDetails(cfg config.Config) map[string]string {
	return gormescli.ConfiguredChannelCapabilityDetails(cfg)
}

func ConfiguredWhatsAppGatewayStatusDetail() string {
	return gormescli.ConfiguredWhatsAppGatewayStatusDetail()
}
