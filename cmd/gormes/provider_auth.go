package main

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func configuredProviderAuthPresent(cfg config.Config) bool {
	if strings.TrimSpace(cfg.Hermes.APIKey) != "" {
		return true
	}
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		return false
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return false
	}
	return pool.RedactedStatus().Count > 0
}
