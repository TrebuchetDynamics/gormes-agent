package router

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type providerCredentialResolution struct {
	Status    CredentialStatus
	Value     string
	Available bool
}

func resolveProviderCredential(route Route, lookupEnv func(string) (string, bool)) providerCredentialResolution {
	resolution := config.ResolveProviderCredential(config.ProviderCredentialRequest{
		Provider:  route.Provider,
		APIKey:    route.apiKey,
		APIKeyEnv: route.APIKeyEnv,
		APIKeyRef: route.APIKeyRef,
		Local:     route.Local,
		Optional:  route.Optional,
		LookupEnv: lookupEnv,
	})
	return providerCredentialResolution{
		Status:    routerCredentialStatus(resolution.Status),
		Value:     strings.TrimSpace(resolution.Value),
		Available: resolution.Available,
	}
}

func routerCredentialStatus(status config.ProviderCredentialStatus) CredentialStatus {
	switch status {
	case config.ProviderCredentialConfigured:
		return CredentialConfigured
	case config.ProviderCredentialNotRequired:
		return CredentialNotNeeded
	default:
		return CredentialMissing
	}
}
