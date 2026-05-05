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
	if strings.EqualFold(provider, config.CodexOAuthProvider) {
		status, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).CheckAuth()
		return err == nil && status.Authenticated
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return false
	}
	for _, entry := range pool.Entries() {
		if pooledCredentialHasUsableAuth(entry) {
			return true
		}
	}
	return false
}

func pooledCredentialHasUsableAuth(entry config.PooledCredential) bool {
	return strings.TrimSpace(entry.AccessToken) != "" ||
		strings.TrimSpace(entry.RefreshToken) != "" ||
		strings.TrimSpace(entry.AgentKey) != ""
}
