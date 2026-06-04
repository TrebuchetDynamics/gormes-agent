package providerauth

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestConfiguredProviderAPIKeyRefPresentFalseWhenUnset(t *testing.T) {
	if ConfiguredProviderAPIKeyRefPresent(config.Config{}) {
		t.Fatal("API key ref unexpectedly present for zero config")
	}
}

func TestConfiguredProviderAPIKeyRefPresentResolvesEnvSecret(t *testing.T) {
	t.Setenv("CUSTOM_PROVIDER_SECRET", "sk-test")
	cfg := config.Config{}
	cfg.Hermes.Provider = "custom"
	cfg.Hermes.APIKeyRef = &config.SecretRef{Source: "env", ID: "CUSTOM_PROVIDER_SECRET"}

	if !ConfiguredProviderAPIKeyRefPresent(cfg) {
		t.Fatal("API key ref not present for available env secret")
	}
}

func TestConfiguredProviderAuthPresentDelegatesConfiguredProviderState(t *testing.T) {
	cfg := config.Config{}
	cfg.Hermes.APIKey = "sk-test"
	if !ConfiguredProviderAuthPresent(cfg) {
		t.Fatal("provider auth not present for configured API key")
	}
}
