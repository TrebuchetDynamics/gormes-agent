package router

import (
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type providerCredentialResolution struct {
	Status    CredentialStatus
	Value     string
	Available bool
}

func resolveProviderCredential(route Route, lookupEnv func(string) (string, bool)) providerCredentialResolution {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if route.Local && route.Optional && strings.TrimSpace(route.apiKey) == "" && strings.TrimSpace(route.APIKeyEnv) == "" && route.APIKeyRef == nil {
		return providerCredentialResolution{Status: CredentialNotNeeded, Available: true}
	}
	if env := strings.TrimSpace(route.APIKeyEnv); env != "" {
		value, ok := lookupEnv(env)
		value = strings.TrimSpace(value)
		if ok && value != "" {
			return providerCredentialResolution{Status: CredentialConfigured, Value: value, Available: true}
		}
		return providerCredentialResolution{Status: CredentialMissing}
	}
	if route.APIKeyRef != nil {
		return resolveSecretRefCredential(*route.APIKeyRef, lookupEnv)
	}
	if value := strings.TrimSpace(route.apiKey); value != "" {
		return providerCredentialResolution{Status: CredentialConfigured, Value: value, Available: true}
	}
	if value, ok := providerManifestCredential(route.Provider, lookupEnv); ok {
		return providerCredentialResolution{Status: CredentialConfigured, Value: value, Available: true}
	}
	return providerCredentialResolution{Status: CredentialMissing}
}

func resolveSecretRefCredential(ref config.SecretRef, lookupEnv func(string) (string, bool)) providerCredentialResolution {
	if strings.EqualFold(string(ref.Source), string(config.SecretRefSourceEnv)) {
		value, ok := lookupEnv(strings.TrimSpace(ref.ID))
		value = strings.TrimSpace(value)
		if ok && value != "" {
			return providerCredentialResolution{Status: CredentialConfigured, Value: value, Available: true}
		}
		return providerCredentialResolution{Status: CredentialMissing}
	}
	// File and future secret providers are redacted handles for the read model.
	// The HTTP upstream adapter does not open them in this slice.
	if strings.TrimSpace(ref.ID) != "" {
		return providerCredentialResolution{Status: CredentialConfigured}
	}
	return providerCredentialResolution{Status: CredentialMissing}
}

func providerManifestCredential(provider string, lookupEnv func(string) (string, bool)) (string, bool) {
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	if !ok {
		return "", false
	}
	for _, env := range entry.EnvVars {
		value, ok := lookupEnv(env)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}
