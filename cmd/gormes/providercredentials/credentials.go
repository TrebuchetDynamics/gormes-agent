package providercredentials

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// Resolve returns the Hermes-compatible endpoint and API key for a provider.
func Resolve(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return ResolveWithHome(cfg, provider, "")
}

// ResolveWithHome returns provider credentials, preferring a scoped credential
// home and falling back to the global pool when the scoped pool is empty.
func ResolveWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	endpoint = strings.TrimSpace(cfg.Hermes.Endpoint)
	apiKey = strings.TrimSpace(cfg.Hermes.APIKey)
	provider = NormalizeName(provider)
	if provider == "openrouter" || llm.IsOpenRouterBaseURL(endpoint) {
		if provider != "" && (endpoint == "" || apiKey == "") {
			credential, evidence, err := selectCredential(provider, credentialHome)
			if err != nil && endpoint == "" && apiKey == "" {
				return "", "", setupError(provider, evidence.Code)
			}
			if err == nil && credential != nil {
				endpoint = firstNonEmpty(endpoint, credentialEndpoint(*credential))
				apiKey = firstNonEmpty(apiKey, credentialAPIKey(*credential))
			}
		}
		runtime := llm.ResolveOpenRouterRuntime(llm.OpenRouterRuntimeRequest{
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
		return resolveCodexOAuth(endpoint, apiKey, credentialHome)
	}
	if provider != "" && (endpoint == "" || apiKey == "") {
		credential, evidence, err := selectCredential(provider, credentialHome)
		if err != nil {
			if endpoint != "" {
				return endpoint, apiKey, nil
			}
			return "", "", setupError(provider, evidence.Code)
		}
		if credential != nil {
			endpoint = firstNonEmpty(endpoint, credentialEndpoint(*credential))
			apiKey = firstNonEmpty(apiKey, credentialAPIKey(*credential))
			if endpoint == "" {
				return "", "", setupError(provider, "credential_missing_inference_base_url")
			}
			if apiKey == "" {
				return "", "", setupError(provider, "credential_missing_access_token")
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

// UsesAnthropicMessages reports whether a provider uses Anthropic's Messages API transport.
func UsesAnthropicMessages(provider string) bool {
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	return ok && entry.TransportFamily == "anthropic_messages"
}

// NormalizeAnthropicMessagesEndpoint trims OpenAI-style /v1 suffixes for Anthropic clients.
func NormalizeAnthropicMessagesEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	return strings.TrimSuffix(endpoint, "/v1")
}

// NormalizeName canonicalizes provider aliases used by credential pools.
func NormalizeName(provider string) string {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(provider)), "_", "-")
	switch normalized {
	case "or", "open-router", "openrouter-free":
		return "openrouter"
	case "novita-ai", "novitaai":
		return "novita"
	case "google", "google-ai-studio", "google-gemini":
		return "gemini"
	default:
		return normalized
	}
}

func resolveCodexOAuth(endpoint, apiKey, credentialHome string) (string, string, error) {
	if apiKey != "" {
		if endpoint == "" {
			return "", "", fmt.Errorf("%s endpoint unconfigured: set [hermes].endpoint or GORMES_ENDPOINT", config.CodexOAuthProvider)
		}
		return endpoint, apiKey, nil
	}
	credential, evidence, err := selectCredential(config.CodexOAuthProvider, credentialHome)
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

func selectCredential(provider, credentialHome string) (*config.PooledCredential, config.CredentialPoolEvidence, error) {
	pool, evidence, err := loadCredentialPool(provider, credentialHome)
	if err != nil {
		return nil, evidence, err
	}
	credential, selection := pool.Select()
	if credential == nil && shouldFallbackCredentialHome(credentialHome, evidence) {
		pool, evidence, err = loadCredentialPool(provider, "")
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

func loadCredentialPool(provider, credentialHome string) (*config.CredentialPool, config.CredentialPoolEvidence, error) {
	return config.LoadCredentialPool(config.CredentialPoolOptions{HermesHome: credentialHome, Provider: provider})
}

func shouldFallbackCredentialHome(credentialHome string, evidence config.CredentialPoolEvidence) bool {
	return strings.TrimSpace(credentialHome) != "" && evidence.Code == config.CredentialPoolEvidenceEmpty
}

func credentialEndpoint(credential config.PooledCredential) string {
	return strings.TrimRight(strings.TrimSpace(firstNonEmpty(credential.InferenceBaseURL, credential.BaseURL)), "/")
}

func credentialAPIKey(credential config.PooledCredential) string {
	return strings.TrimSpace(firstNonEmpty(credential.AccessToken, credential.AgentKey))
}

func setupError(provider, reason string) error {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
