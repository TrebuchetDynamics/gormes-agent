package main

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

var providerPool = gormescli.NewProviderClientPool()

func getOrCreateProviderClient(cfg config.Config, providerName string) (llm.Client, error) {
	return providerPool.GetOrCreate(cfg, providerName)
}

func resetProviderPoolForTesting() {
	providerPool.Reset()
}

func newProviderHTTPClient(cfg config.Config, provider string) (llm.Client, error) {
	return gormescli.NewProviderHTTPClient(cfg, provider)
}

func resolveProviderHTTPClientCredentials(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return gormescli.ResolveProviderHTTPClientCredentials(cfg, provider)
}

func newProviderHTTPClientWithCredentialHome(cfg config.Config, provider, credentialHome string) (llm.Client, error) {
	return gormescli.NewProviderHTTPClientWithCredentialHome(cfg, provider, credentialHome)
}

func providerUsesAnthropicMessages(provider string) bool {
	return gormescli.ProviderUsesAnthropicMessages(provider)
}

func normalizeAnthropicMessagesEndpoint(endpoint string) string {
	return gormescli.NormalizeAnthropicMessagesEndpoint(endpoint)
}

func resolveProviderHTTPClientCredentialsWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	return gormescli.ResolveProviderHTTPClientCredentialsWithHome(cfg, provider, credentialHome)
}

func normalizeProviderName(provider string) string {
	return gormescli.NormalizeProviderName(provider)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
