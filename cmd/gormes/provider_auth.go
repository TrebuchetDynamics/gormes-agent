package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func configuredProviderAuthPresent(cfg config.Config) bool {
	return gormescli.ConfiguredProviderAuthPresent(cfg)
}

func configuredProviderAPIKeyRefPresent(cfg config.Config) bool {
	return gormescli.ConfiguredProviderAPIKeyRefPresent(cfg)
}
