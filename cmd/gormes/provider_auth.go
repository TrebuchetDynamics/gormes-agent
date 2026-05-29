package main

import "github.com/TrebuchetDynamics/gormes-agent/internal/config"

func configuredProviderAuthPresent(cfg config.Config) bool {
	return config.ConfiguredProviderAuthPresent(cfg)
}

func configuredProviderAPIKeyRefPresent(cfg config.Config) bool {
	if cfg.Hermes.APIKeyRef == nil {
		return false
	}
	return config.ResolveProviderCredential(config.ProviderCredentialRequest{
		Provider:  cfg.Hermes.Provider,
		APIKeyRef: cfg.Hermes.APIKeyRef,
		Secrets:   cfg.Secrets,
	}).Available
}
