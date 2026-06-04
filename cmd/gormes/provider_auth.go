package main

import (
	providerauthcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/providerauth"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func configuredProviderAuthPresent(cfg config.Config) bool {
	return providerauthcmd.ConfiguredProviderAuthPresent(cfg)
}

func configuredProviderAPIKeyRefPresent(cfg config.Config) bool {
	return providerauthcmd.ConfiguredProviderAPIKeyRefPresent(cfg)
}
