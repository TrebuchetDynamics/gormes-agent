package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/provider"
)

// providerPool caches constructed provider clients so the oneshot cold-start
// path only constructs the selected provider. Gateway and doctor paths also
// benefit from reuse, but the pool's primary design goal is avoiding eager
// construction of unselected providers during `gormes -z`.
var providerPool = provider.NewClientPool()

// getOrCreateProviderClient returns a hermes.Client for the given provider,
// constructing it on first access via newProviderHTTPClient. Subsequent
// calls for the same provider return the cached instance.
func getOrCreateProviderClient(cfg config.Config, providerName string) (hermes.Client, error) {
	providerPool.Register(providerName, func() (hermes.Client, error) {
		client, err := newProviderHTTPClient(cfg, providerName)
		if err != nil {
			return nil, err
		}
		// Wire credential exhaustion callback: on 429/401, mark the credential
		// exhausted in the pool and invalidate the cached client so the next
		// request selects a fresh credential.
		normalized := normalizeProviderName(providerName)
		hermes.SetOnCredentialExhausted(client, func(statusCode int, reason string, _ http.Header) {
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
		if provider != "" && (endpoint == "" || apiKey == "") {
			credential, evidence, err := selectProviderCredential(provider, credentialHome)
			if err != nil && endpoint == "" && apiKey == "" {
				return "", "", providerCredentialSetupError(provider, evidence.Code)
			}
			if err == nil && credential != nil {
				endpoint = firstNonEmpty(endpoint, credentialEndpoint(*credential))
				apiKey = firstNonEmpty(apiKey, credentialAPIKey(*credential))
			}
		}
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
	if provider != "" && (endpoint == "" || apiKey == "") {
		credential, evidence, err := selectProviderCredential(provider, credentialHome)
		if err != nil {
			if endpoint != "" {
				return endpoint, apiKey, nil
			}
			return "", "", providerCredentialSetupError(provider, evidence.Code)
		}
		if credential != nil {
			endpoint = firstNonEmpty(endpoint, credentialEndpoint(*credential))
			apiKey = firstNonEmpty(apiKey, credentialAPIKey(*credential))
			if endpoint == "" {
				return "", "", providerCredentialSetupError(provider, "credential_missing_inference_base_url")
			}
			if apiKey == "" {
				return "", "", providerCredentialSetupError(provider, "credential_missing_access_token")
			}
			return endpoint, apiKey, nil
		}
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
	credential, evidence, err := selectProviderCredential(config.CodexOAuthProvider, credentialHome)
	if err != nil {
		return "", "", codexOAuthSetupError(evidence.Code)
	}
	if credential == nil {
		return "", "", codexOAuthSetupError(evidence.Code)
	}
	resolvedEndpoint := strings.TrimRight(strings.TrimSpace(firstNonEmpty(endpoint, credentialEndpoint(*credential))), "/")
	resolvedAPIKey := strings.TrimSpace(firstNonEmpty(apiKey, credentialAPIKey(*credential)))
	if resolvedEndpoint == "" {
		return "", "", codexOAuthSetupError("credential_missing_inference_base_url")
	}
	if resolvedAPIKey == "" {
		return "", "", codexOAuthSetupError("credential_missing_access_token")
	}
	return resolvedEndpoint, resolvedAPIKey, nil
}

func selectProviderCredential(provider, credentialHome string) (*config.PooledCredential, config.CredentialPoolEvidence, error) {
	pool, evidence, err := loadProviderCredentialPool(provider, credentialHome)
	if err != nil {
		return nil, evidence, err
	}
	credential, selection := pool.Select()
	if credential == nil && shouldFallbackProviderCredentialHome(credentialHome, evidence) {
		pool, evidence, err = loadProviderCredentialPool(provider, "")
		if err != nil {
			return nil, evidence, err
		}
		credential, selection = pool.Select()
	}
	if credential == nil {
		return nil, selection, nil
	}
	return credential, evidence, nil
}

func loadProviderCredentialPool(provider, credentialHome string) (*config.CredentialPool, config.CredentialPoolEvidence, error) {
	return config.LoadCredentialPool(config.CredentialPoolOptions{HermesHome: credentialHome, Provider: provider})
}

func shouldFallbackProviderCredentialHome(credentialHome string, evidence config.CredentialPoolEvidence) bool {
	return strings.TrimSpace(credentialHome) != "" && evidence.Code == config.CredentialPoolEvidenceEmpty
}

func credentialEndpoint(credential config.PooledCredential) string {
	return strings.TrimRight(strings.TrimSpace(firstNonEmpty(credential.InferenceBaseURL, credential.BaseURL)), "/")
}

func credentialAPIKey(credential config.PooledCredential) string {
	return strings.TrimSpace(firstNonEmpty(credential.AccessToken, credential.AgentKey))
}

func providerCredentialSetupError(provider, reason string) error {
	if provider == config.CodexOAuthProvider {
		return codexOAuthSetupError(reason)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "credential_unavailable"
	}
	return fmt.Errorf("%s credential unavailable: run `gormes auth add %s` (status=%s)", provider, provider, reason)
}

func codexOAuthSetupError(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "credential_unavailable"
	}
	return fmt.Errorf("%s credential unavailable: run `gormes auth add %s --type oauth` (status=%s)", config.CodexOAuthProvider, config.CodexOAuthProvider, reason)
}

func normalizeProviderName(provider string) string {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(provider)), "_", "-")
	switch normalized {
	case "novita-ai", "novitaai":
		return "novita"
	default:
		return normalized
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
