package main

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func newProviderHTTPClient(cfg config.Config, provider string) (hermes.Client, error) {
	endpoint, apiKey, err := resolveProviderHTTPClientCredentials(cfg, provider)
	if err != nil {
		return nil, err
	}
	if providerUsesAnthropicMessages(provider) {
		return hermes.NewAnthropicClient(normalizeAnthropicMessagesEndpoint(endpoint), apiKey), nil
	}
	return hermes.NewHTTPClientWithProvider(endpoint, apiKey, provider), nil
}

func resolveProviderHTTPClientCredentials(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return resolveProviderHTTPClientCredentialsWithHome(cfg, provider, "")
}

func newProviderHTTPClientWithCredentialHome(cfg config.Config, provider, credentialHome string) (hermes.Client, error) {
	endpoint, apiKey, err := resolveProviderHTTPClientCredentialsWithHome(cfg, provider, credentialHome)
	if err != nil {
		return nil, err
	}
	if providerUsesAnthropicMessages(provider) {
		return hermes.NewAnthropicClient(normalizeAnthropicMessagesEndpoint(endpoint), apiKey), nil
	}
	return hermes.NewHTTPClientWithProvider(endpoint, apiKey, provider), nil
}

func providerUsesAnthropicMessages(provider string) bool {
	entry, ok := hermes.ResolveProviderManifestEntry(provider)
	return ok && entry.TransportFamily == "anthropic_messages"
}

func normalizeAnthropicMessagesEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	return strings.TrimSuffix(endpoint, "/v1")
}

func resolveProviderHTTPClientCredentialsWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	endpoint = strings.TrimSpace(cfg.Hermes.Endpoint)
	apiKey = strings.TrimSpace(cfg.Hermes.APIKey)
	provider = normalizeProviderName(provider)
	if provider == "openrouter" || hermes.IsOpenRouterBaseURL(endpoint) {
		runtime := hermes.ResolveOpenRouterRuntime(hermes.OpenRouterRuntimeRequest{
			Provider: provider,
			BaseURL:  endpoint,
			APIKey:   apiKey,
		})
		if runtime.MissingAPIKey {
			return "", "", fmt.Errorf("openrouter credential unavailable: set OPENROUTER_API_KEY, OPENAI_API_KEY, or [hermes].api_key")
		}
		return runtime.BaseURL, runtime.APIKey, nil
	}
	if provider == config.CodexOAuthProvider {
		return resolveCodexOAuthHTTPClientCredentials(endpoint, apiKey, credentialHome)
	}
	if endpoint != "" {
		return endpoint, apiKey, nil
	}

	// No explicit endpoint. Providers must surface a clear setup error
	// rather than build an HTTP client with an empty base URL — which would
	// otherwise emit `Post "/v1/responses": unsupported protocol scheme ""`
	// at the first turn.
	if provider == "" {
		return "", "", fmt.Errorf("hermes endpoint unconfigured and no provider declared: set [hermes].endpoint or GORMES_ENDPOINT, or configure a [hermes].provider with credential support")
	}
	return "", "", fmt.Errorf("hermes endpoint unconfigured for provider %q: set [hermes].endpoint or GORMES_ENDPOINT to a Hermes-compatible base URL (gormes does not yet ship a native runtime for %q)", provider, provider)
}

func resolveCodexOAuthHTTPClientCredentials(endpoint, apiKey, credentialHome string) (string, string, error) {
	if apiKey != "" {
		if endpoint == "" {
			return "", "", fmt.Errorf("%s endpoint unconfigured: set [hermes].endpoint or GORMES_ENDPOINT", config.CodexOAuthProvider)
		}
		return endpoint, apiKey, nil
	}
	pool, evidence, err := config.LoadCredentialPool(config.CredentialPoolOptions{HermesHome: credentialHome, Provider: config.CodexOAuthProvider})
	if err != nil {
		return "", "", fmt.Errorf("%s credential pool unavailable: %s", config.CodexOAuthProvider, evidence.Code)
	}
	credential, selection := pool.Select()
	if credential == nil {
		return "", "", fmt.Errorf("%s credential unavailable: %s", config.CodexOAuthProvider, selection.Code)
	}
	resolvedEndpoint := strings.TrimRight(strings.TrimSpace(firstNonEmpty(endpoint, credential.InferenceBaseURL, credential.BaseURL)), "/")
	resolvedAPIKey := strings.TrimSpace(credential.AccessToken)
	if resolvedEndpoint == "" {
		return "", "", fmt.Errorf("%s credential missing inference base URL", config.CodexOAuthProvider)
	}
	if resolvedAPIKey == "" {
		return "", "", fmt.Errorf("%s credential missing access token", config.CodexOAuthProvider)
	}
	return resolvedEndpoint, resolvedAPIKey, nil
}

func normalizeProviderName(provider string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(provider)), "_", "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
