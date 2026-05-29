package config

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/auth"
)

type ProviderCredentialStatus = auth.ProviderCredentialStatus

const (
	ProviderCredentialConfigured  = auth.ProviderCredentialConfigured
	ProviderCredentialMissing     = auth.ProviderCredentialMissing
	ProviderCredentialNotRequired = auth.ProviderCredentialNotRequired
)

type ProviderCredentialSource = auth.ProviderCredentialSource

const (
	ProviderCredentialSourceNone      = auth.ProviderCredentialSourceNone
	ProviderCredentialSourceEnv       = auth.ProviderCredentialSourceEnv
	ProviderCredentialSourceSecretRef = auth.ProviderCredentialSourceSecretRef
	ProviderCredentialSourceInline    = auth.ProviderCredentialSourceInline
	ProviderCredentialSourceManifest  = auth.ProviderCredentialSourceManifest
	ProviderCredentialSourcePool      = auth.ProviderCredentialSourcePool
	ProviderCredentialSourceCodex     = auth.ProviderCredentialSourceCodex
)

type ProviderCredentialRequest = auth.ProviderCredentialRequest
type ProviderCredentialResolution = auth.ProviderCredentialResolution

func ResolveProviderCredential(req ProviderCredentialRequest) ProviderCredentialResolution {
	return auth.ResolveProviderCredential(req)
}

// ConfiguredProviderAuthPresent reports whether cfg has usable provider auth
// through the same credential resolution seam used by runtime callers.
func ConfiguredProviderAuthPresent(cfg Config) bool {
	resolution := ResolveProviderCredential(ProviderCredentialRequest{
		Provider:          cfg.Hermes.Provider,
		APIKey:            cfg.Hermes.APIKey,
		APIKeyRef:         cfg.Hermes.APIKeyRef,
		Secrets:           cfg.Secrets,
		UseCredentialPool: strings.TrimSpace(cfg.Hermes.Provider) != "",
		UseCodexOAuth:     true,
	})
	return resolution.Available
}
