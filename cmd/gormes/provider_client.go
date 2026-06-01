package main

import (
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/providercredentials"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/provider"
)

// providerPool caches constructed provider clients so the oneshot cold-start
// path only constructs the selected provider. Gateway and doctor paths also
// benefit from reuse, but the pool's primary design goal is avoiding eager
// construction of unselected providers during scripted chat.
var providerPool = provider.NewClientPool()

// getOrCreateProviderClient returns a llm.Client for the given provider,
// constructing it on first access via newProviderHTTPClient. Subsequent
// calls for the same provider return the cached instance.
func getOrCreateProviderClient(cfg config.Config, providerName string) (llm.Client, error) {
	providerPool.Register(providerName, func() (llm.Client, error) {
		client, err := newProviderHTTPClient(cfg, providerName)
		if err != nil {
			return nil, err
		}
		// Wire credential exhaustion callback: on 429/401, mark the credential
		// exhausted in the pool and invalidate the cached client so the next
		// request selects a fresh credential.
		normalized := normalizeProviderName(providerName)
		llm.SetOnCredentialExhausted(client, func(statusCode int, reason string, _ http.Header) {
			pool, _, loadErr := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: normalized})
			if loadErr != nil {
				return
			}
			pool.MarkExhaustedAndRotate(config.CredentialExhaustion{
				StatusCode: statusCode,
				Reason:     reason,
			})
			providerPool.Invalidate(normalized)
		})
		return client, nil
	})
	return providerPool.Get(providerName)
}

// resetProviderPoolForTesting clears all cached clients. Only exported for
// test use; must not be called from production code paths.
func resetProviderPoolForTesting() {
	providerPool.Reset()
}

func newProviderHTTPClient(cfg config.Config, provider string) (llm.Client, error) {
	endpoint, apiKey, err := resolveProviderHTTPClientCredentials(cfg, provider)
	if err != nil {
		return nil, err
	}
	if providerUsesAnthropicMessages(provider) {
		return llm.NewAnthropicClient(normalizeAnthropicMessagesEndpoint(endpoint), apiKey), nil
	}
	return llm.NewHTTPClientWithProvider(endpoint, apiKey, provider), nil
}

func resolveProviderHTTPClientCredentials(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return providercredentials.Resolve(cfg, provider)
}

func newProviderHTTPClientWithCredentialHome(cfg config.Config, provider, credentialHome string) (llm.Client, error) {
	endpoint, apiKey, err := resolveProviderHTTPClientCredentialsWithHome(cfg, provider, credentialHome)
	if err != nil {
		return nil, err
	}
	if providerUsesAnthropicMessages(provider) {
		return llm.NewAnthropicClient(normalizeAnthropicMessagesEndpoint(endpoint), apiKey), nil
	}
	return llm.NewHTTPClientWithProvider(endpoint, apiKey, provider), nil
}

func providerUsesAnthropicMessages(provider string) bool {
	return providercredentials.UsesAnthropicMessages(provider)
}

func normalizeAnthropicMessagesEndpoint(endpoint string) string {
	return providercredentials.NormalizeAnthropicMessagesEndpoint(endpoint)
}

func resolveProviderHTTPClientCredentialsWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	return providercredentials.ResolveWithHome(cfg, provider, credentialHome)
}

func normalizeProviderName(provider string) string {
	return providercredentials.NormalizeName(provider)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
