package main

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func configuredMaxToolIterations(cfg config.Config) int {
	if cfg.Runtime.MaxToolIterations > 0 {
		return cfg.Runtime.MaxToolIterations
	}
	return kernel.DefaultMaxToolIterations
}

func configuredTTSProvider(cfg config.Config) string {
	provider := strings.ToLower(strings.TrimSpace(cfg.Runtime.TTSProvider))
	if provider == "" {
		return "edge"
	}
	return provider
}
