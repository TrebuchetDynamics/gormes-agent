package gormescli

import (
	providerauthapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/providerauth"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func ConfiguredProviderAuthPresent(cfg config.Config) bool {
	return providerauthapp.ConfiguredProviderAuthPresent(cfg)
}

func ConfiguredProviderAPIKeyRefPresent(cfg config.Config) bool {
	return providerauthapp.ConfiguredProviderAPIKeyRefPresent(cfg)
}
