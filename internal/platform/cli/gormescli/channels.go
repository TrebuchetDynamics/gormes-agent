package gormescli

import (
	"os"

	appchannels "github.com/TrebuchetDynamics/gormes-agent/internal/app/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func ConfiguredChannelCapabilityDetails(cfg config.Config) map[string]string {
	return appchannels.ConfiguredCapabilityDetails(cfg, os.Getenv)
}

func ConfiguredWhatsAppGatewayStatusDetail() string {
	return appchannels.ConfiguredWhatsAppGatewayStatusDetail(os.Getenv)
}
