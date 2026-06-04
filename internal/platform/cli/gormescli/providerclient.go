package gormescli

import (
	appproviderclient "github.com/TrebuchetDynamics/gormes-agent/internal/app/providerclient"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type ProviderClientPool = appproviderclient.Pool

func NewProviderClientPool() *ProviderClientPool {
	return appproviderclient.NewPool()
}

func NewProviderHTTPClient(cfg config.Config, provider string) (llm.Client, error) {
	return appproviderclient.NewHTTPClient(cfg, provider)
}

func ResolveProviderHTTPClientCredentials(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return appproviderclient.ResolveCredentials(cfg, provider)
}

func NewProviderHTTPClientWithCredentialHome(cfg config.Config, provider, credentialHome string) (llm.Client, error) {
	return appproviderclient.NewHTTPClientWithCredentialHome(cfg, provider, credentialHome)
}

func ProviderUsesAnthropicMessages(provider string) bool {
	return appproviderclient.UsesAnthropicMessages(provider)
}

func NormalizeAnthropicMessagesEndpoint(endpoint string) string {
	return appproviderclient.NormalizeAnthropicMessagesEndpoint(endpoint)
}

func ResolveProviderHTTPClientCredentialsWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	return appproviderclient.ResolveCredentialsWithHome(cfg, provider, credentialHome)
}

func NormalizeProviderName(provider string) string {
	return appproviderclient.NormalizeProviderName(provider)
}
