package main

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/providerclient"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

var providerPool = providerclient.NewPool()

func getOrCreateProviderClient(cfg config.Config, providerName string) (llm.Client, error) {
	return providerPool.GetOrCreate(cfg, providerName)
}

func resetProviderPoolForTesting() {
	providerPool.Reset()
}

func newProviderHTTPClient(cfg config.Config, provider string) (llm.Client, error) {
	return providerclient.NewHTTPClient(cfg, provider)
}

func resolveProviderHTTPClientCredentials(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return providerclient.ResolveCredentials(cfg, provider)
}

func newProviderHTTPClientWithCredentialHome(cfg config.Config, provider, credentialHome string) (llm.Client, error) {
	return providerclient.NewHTTPClientWithCredentialHome(cfg, provider, credentialHome)
}

func providerUsesAnthropicMessages(provider string) bool {
	return providerclient.UsesAnthropicMessages(provider)
}

func normalizeAnthropicMessagesEndpoint(endpoint string) string {
	return providerclient.NormalizeAnthropicMessagesEndpoint(endpoint)
}

func resolveProviderHTTPClientCredentialsWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	return providerclient.ResolveCredentialsWithHome(cfg, provider, credentialHome)
}

func normalizeProviderName(provider string) string {
	return providerclient.NormalizeProviderName(provider)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
