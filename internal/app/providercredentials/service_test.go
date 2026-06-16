package providercredentials

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestResolveOpenRouterAliasUsesCanonicalCredentialPool(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: "openrouter"}, []config.PooledCredential{{
		ID:               "openrouter-primary",
		Label:            "OpenRouter primary",
		AuthType:         config.CredentialAuthAPIKey,
		Source:           "manual",
		AccessToken:      "or-pool-key",
		InferenceBaseURL: llm.OpenRouterDefaultBaseURL,
		LastStatus:       config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	endpoint, apiKey, err := Resolve(config.Config{Hermes: config.HermesCfg{Provider: "openrouter-free"}}, "openrouter-free")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if endpoint != llm.OpenRouterDefaultBaseURL || apiKey != "or-pool-key" {
		t.Fatalf("endpoint/apiKey = %q/%q, want canonical OpenRouter credential", endpoint, apiKey)
	}
}

func TestResolveMissingProviderEndpointFailsClearly(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	endpoint, apiKey, err := Resolve(config.Config{Hermes: config.HermesCfg{Provider: "anthropic", APIKey: "sk-test"}}, "anthropic")
	if err == nil {
		t.Fatalf("err=nil endpoint/apiKey=%q/%q, want missing endpoint setup error", endpoint, apiKey)
	}
	if got := strings.ToLower(err.Error()); !strings.Contains(got, "endpoint") || !strings.Contains(got, "anthropic") {
		t.Fatalf("error = %q, want provider endpoint guidance", err)
	}
}

func TestResolveWithHomeFallsBackToGlobalPoolWhenScopedPoolEmpty(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: "custom"}, []config.PooledCredential{{
		ID:               "custom-global",
		Label:            "global",
		AuthType:         config.CredentialAuthAPIKey,
		Source:           "manual",
		AccessToken:      "global-token",
		InferenceBaseURL: "https://global.example/v1",
		LastStatus:       config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	endpoint, apiKey, err := ResolveWithHome(config.Config{Hermes: config.HermesCfg{Provider: "custom"}}, "custom", gormesHome+"/agents/main/agent")
	if err != nil {
		t.Fatalf("ResolveWithHome: %v", err)
	}
	if endpoint != "https://global.example/v1" || apiKey != "global-token" {
		t.Fatalf("endpoint/apiKey = %q/%q, want global credential fallback", endpoint, apiKey)
	}
}
