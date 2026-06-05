// Package runtime owns the channel-neutral resolver that decides whether a
// gateway turn should be served by an explicit OpenAI-compatible HTTP / proxy
// client or by a provider-aware native runtime client. The resolver is pure:
// it takes already-resolved provider/model/credential values plus injectable
// factories, returns a Binding describing the decision, and never opens a
// network connection itself. Channel adapters call ResolveBinding through a
// shared seam so the implicit 127.0.0.1:8642 backend is never reached when no
// explicit endpoint is configured.
package runtime

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// EndpointSource records which input shaped the binding so status evidence
// can name it without echoing secrets.
type EndpointSource string

const (
	// EndpointSourceUnconfigured marks a binding that could not be built —
	// no explicit endpoint and no complete native provider config.
	EndpointSourceUnconfigured EndpointSource = "unconfigured"
	// EndpointSourceExplicitEndpoint marks an OpenAI-compatible HTTP path
	// driven by a config-provided endpoint or proxy URL.
	EndpointSourceExplicitEndpoint EndpointSource = "explicit_endpoint"
	// EndpointSourceNativeProvider marks a provider-aware native runtime
	// client built from Hermes YAML model.provider/model.default.
	EndpointSourceNativeProvider EndpointSource = "native_provider"
)

// Degraded reason codes are stable identifiers safe to surface in status
// evidence. They never include secrets, raw URLs, or process-local
// implementation details.
const (
	DegradedReasonProviderConfigMissing    = "provider_config_missing"
	DegradedReasonNativeRuntimeUnavailable = "native_runtime_unavailable"
)

// HTTPClientFactory builds an OpenAI-compatible / proxy HTTP client. The
// concrete factory lives in cmd/gormes; tests inject fakes through this seam.
type HTTPClientFactory func(baseURL, apiKey, provider string) llm.Client

// NativeClientFactory builds a provider-aware native runtime client. Tests
// inject fakes; production wiring lands in a follow-up slice that brings the
// channel-neutral factory into cmd/gormes without re-introducing the implicit
// localhost backend.
type NativeClientFactory func(req NativeClientRequest) (llm.Client, error)

// NativeClientRequest carries the already-resolved provider/model/credential
// values into the native client factory. It deliberately does not carry raw
// configuration objects — the resolver decides what reaches the seam.
type NativeClientRequest struct {
	Provider string
	Model    string
	APIKey   string
}

// BindingRequest is the channel-neutral resolver input. Channel adapters
// build it from already-resolved config + flag values and never inject
// channel-specific conditionals.
type BindingRequest struct {
	Provider string
	Model    string
	APIKey   string
	Endpoint string
	ProxyURL string
	Channel  string

	HTTPClientFactory   HTTPClientFactory
	NativeClientFactory NativeClientFactory
}

// Binding is the resolver output. EndpointSource and DegradedReasons are
// stable identifiers safe to log; APIKey stays inside the struct only so the
// caller can hand it to the chosen client and is intentionally redacted by
// StatusEvidence.
type Binding struct {
	Provider        string
	Model           string
	APIKey          string
	Endpoint        string
	EndpointSource  EndpointSource
	DegradedReasons []string
	Client          llm.Client
}

// ResolveBinding picks the explicit OpenAI-compatible / proxy path when an
// endpoint or proxy URL is configured, and otherwise builds a provider-aware
// native runtime client from Hermes YAML model config. If the native path is
// requested but provider/model/api-key resolution is incomplete, the binding
// records degraded evidence and leaves Client nil so callers can surface
// provider_config_missing / native_runtime_unavailable to operators without
// silently dialling 127.0.0.1:8642.
func ResolveBinding(req BindingRequest) (Binding, error) {
	provider := strings.TrimSpace(req.Provider)
	model := strings.TrimSpace(req.Model)
	apiKey := strings.TrimSpace(req.APIKey)
	endpoint := strings.TrimSpace(req.Endpoint)
	proxyURL := strings.TrimSpace(req.ProxyURL)

	binding := Binding{
		Provider: provider,
		Model:    model,
		APIKey:   apiKey,
	}

	explicit := endpoint
	if explicit == "" {
		explicit = proxyURL
	}
	if explicit != "" {
		binding.Endpoint = explicit
		binding.EndpointSource = EndpointSourceExplicitEndpoint
		if req.HTTPClientFactory != nil {
			binding.Client = req.HTTPClientFactory(explicit, apiKey, provider)
		}
		return binding, nil
	}

	if provider == "" || model == "" || apiKey == "" {
		binding.EndpointSource = EndpointSourceUnconfigured
		binding.DegradedReasons = []string{
			DegradedReasonProviderConfigMissing,
			DegradedReasonNativeRuntimeUnavailable,
		}
		return binding, nil
	}

	binding.EndpointSource = EndpointSourceNativeProvider
	if req.NativeClientFactory != nil {
		client, err := req.NativeClientFactory(NativeClientRequest{
			Provider: provider,
			Model:    model,
			APIKey:   apiKey,
		})
		if err != nil {
			binding.EndpointSource = EndpointSourceUnconfigured
			binding.DegradedReasons = []string{DegradedReasonNativeRuntimeUnavailable}
			binding.Client = nil
			return binding, nil
		}
		binding.Client = client
	}
	return binding, nil
}

// StatusEvidence returns the redacted status map gateway adapters publish so
// operators can see provider, model, endpoint_source, and degraded_reasons
// without ever exposing api keys, proxy keys, or raw secret-bearing URLs.
func StatusEvidence(b Binding) map[string]any {
	status := map[string]any{
		"provider":        b.Provider,
		"model":           b.Model,
		"endpoint_source": string(b.EndpointSource),
	}
	if len(b.DegradedReasons) > 0 {
		reasons := make([]string, len(b.DegradedReasons))
		copy(reasons, b.DegradedReasons)
		status["degraded_reasons"] = reasons
	}
	return status
}
