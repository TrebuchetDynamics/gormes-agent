package providerclient

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/providercredentials"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/provider"
)

// Pool caches constructed provider clients so the oneshot cold-start path only
// constructs the selected provider. Gateway and doctor paths also benefit from
// reuse, but the pool's primary design goal is avoiding eager construction of
// unselected providers during scripted chat.
type Pool struct {
	clients *provider.ClientPool
}

func NewPool() *Pool {
	return &Pool{clients: provider.NewClientPool()}
}

func (p *Pool) GetOrCreate(cfg config.Config, providerName string) (llm.Client, error) {
	p.clients.Register(providerName, func() (llm.Client, error) {
		client, err := NewHTTPClient(cfg, providerName)
		if err != nil {
			return nil, err
		}
		// Wire credential exhaustion callback: on 429/401, mark the credential
		// exhausted in the pool and invalidate the cached client so the next
		// request selects a fresh credential.
		normalized := NormalizeProviderName(providerName)
		llm.SetOnCredentialExhausted(client, func(statusCode int, reason string, _ http.Header) {
			pool, _, loadErr := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: normalized})
			if loadErr != nil {
				return
			}
			pool.MarkExhaustedAndRotate(config.CredentialExhaustion{
				StatusCode: statusCode,
				Reason:     reason,
			})
			p.clients.Invalidate(normalized)
		})
		return client, nil
	})
	return p.clients.Get(providerName)
}

func (p *Pool) Reset() {
	p.clients.Reset()
}

func NewHTTPClient(cfg config.Config, provider string) (llm.Client, error) {
	if NormalizeProviderName(provider) == "gollmfree" {
		return llm.NewGollmfreeClient(), nil
	}
	endpoint, apiKey, err := ResolveCredentials(cfg, provider)
	if err != nil {
		return nil, err
	}
	if UsesAnthropicMessages(provider) {
		return llm.NewAnthropicClient(NormalizeAnthropicMessagesEndpoint(endpoint), apiKey), nil
	}
	return llm.NewHTTPClientWithProvider(endpoint, apiKey, provider), nil
}

func ResolveCredentials(cfg config.Config, provider string) (endpoint, apiKey string, err error) {
	return providercredentials.Resolve(cfg, provider)
}

func NewHTTPClientWithCredentialHome(cfg config.Config, provider, credentialHome string) (llm.Client, error) {
	if NormalizeProviderName(provider) == "gollmfree" {
		return llm.NewGollmfreeClient(), nil
	}
	endpoint, apiKey, err := ResolveCredentialsWithHome(cfg, provider, credentialHome)
	if err != nil {
		return nil, err
	}
	if UsesAnthropicMessages(provider) {
		return llm.NewAnthropicClient(NormalizeAnthropicMessagesEndpoint(endpoint), apiKey), nil
	}
	return llm.NewHTTPClientWithProvider(endpoint, apiKey, provider), nil
}

func UsesAnthropicMessages(provider string) bool {
	return providercredentials.UsesAnthropicMessages(provider)
}

func NormalizeAnthropicMessagesEndpoint(endpoint string) string {
	return providercredentials.NormalizeAnthropicMessagesEndpoint(endpoint)
}

func ResolveCredentialsWithHome(cfg config.Config, provider, credentialHome string) (endpoint, apiKey string, err error) {
	return providercredentials.ResolveWithHome(cfg, provider, credentialHome)
}

func NormalizeProviderName(provider string) string {
	return providercredentials.NormalizeName(provider)
}
