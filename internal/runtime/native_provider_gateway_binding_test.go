package runtime_test

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
)

// fakeNativeClient stands in for a provider-aware native runtime client. It
// captures the provider/model/api-key seam so the resolver can be exercised
// without dialling a real backend.
type fakeNativeClient struct {
	provider string
	model    string
	apiKey   string
}

func (fakeNativeClient) OpenStream(context.Context, hermes.ChatRequest) (hermes.Stream, error) {
	return nil, nil
}
func (fakeNativeClient) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, nil
}
func (fakeNativeClient) Health(context.Context) error { return nil }

// fakeHTTPClient stands in for the OpenAI-compatible / proxy HTTP client.
type fakeHTTPClient struct {
	baseURL  string
	apiKey   string
	provider string
}

func (fakeHTTPClient) OpenStream(context.Context, hermes.ChatRequest) (hermes.Stream, error) {
	return nil, nil
}
func (fakeHTTPClient) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, nil
}
func (fakeHTTPClient) Health(context.Context) error { return nil }

// TestResolveNativeRuntimeBinding_NoEndpointBuildsNativeProviderClient covers
// acceptance bullet 1: with Hermes YAML model.provider/model.default and no
// endpoint/proxy configured, a gateway turn must build a provider-aware
// native runtime client and must NOT attempt 127.0.0.1:8642.
func TestResolveNativeRuntimeBinding_NoEndpointBuildsNativeProviderClient(t *testing.T) {
	t.Parallel()

	httpCalls := 0
	nativeCalls := 0
	var nativeSeen fakeNativeClient

	binding, err := runtime.ResolveBinding(runtime.BindingRequest{
		Provider: "anthropic",
		Model:    "claude-sonnet-latest",
		APIKey:   "secret-key-anthropic",
		Endpoint: "",
		ProxyURL: "",
		HTTPClientFactory: func(baseURL, apiKey, provider string) hermes.Client {
			httpCalls++
			return fakeHTTPClient{baseURL: baseURL, apiKey: apiKey, provider: provider}
		},
		NativeClientFactory: func(req runtime.NativeClientRequest) (hermes.Client, error) {
			nativeCalls++
			nativeSeen = fakeNativeClient{provider: req.Provider, model: req.Model, apiKey: req.APIKey}
			return fakeNativeClient{provider: req.Provider, model: req.Model, apiKey: req.APIKey}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveBinding: %v", err)
	}

	if httpCalls != 0 {
		t.Errorf("HTTPClientFactory called %d times; want 0 — must not dial OpenAI-compatible HTTP", httpCalls)
	}
	if nativeCalls != 1 {
		t.Fatalf("NativeClientFactory called %d times; want 1", nativeCalls)
	}
	if binding.EndpointSource != runtime.EndpointSourceNativeProvider {
		t.Errorf("EndpointSource = %q; want %q", binding.EndpointSource, runtime.EndpointSourceNativeProvider)
	}
	if binding.Endpoint != "" {
		t.Errorf("Endpoint = %q; want empty (no implicit localhost)", binding.Endpoint)
	}
	if strings.Contains(binding.Endpoint, "127.0.0.1:8642") {
		t.Errorf("binding endpoint silently dialled implicit localhost backend: %q", binding.Endpoint)
	}
	if binding.Provider != "anthropic" {
		t.Errorf("Provider = %q; want anthropic", binding.Provider)
	}
	if binding.Model != "claude-sonnet-latest" {
		t.Errorf("Model = %q; want claude-sonnet-latest", binding.Model)
	}
	if nativeSeen.provider != "anthropic" || nativeSeen.model != "claude-sonnet-latest" || nativeSeen.apiKey != "secret-key-anthropic" {
		t.Errorf("native factory request mismatch: %+v", nativeSeen)
	}
	if binding.Client == nil {
		t.Errorf("binding.Client is nil; want native client")
	}
}

// TestResolveNativeRuntimeBinding_ExplicitEndpointPreservesOpenAICompatible
// covers acceptance bullet 2: with an explicit endpoint/proxy URL, gateway
// preserves the OpenAI-compatible/proxy path and records that the endpoint
// was explicit.
func TestResolveNativeRuntimeBinding_ExplicitEndpointPreservesOpenAICompatible(t *testing.T) {
	t.Parallel()

	httpCalls := 0
	nativeCalls := 0
	var seen fakeHTTPClient

	binding, err := runtime.ResolveBinding(runtime.BindingRequest{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		APIKey:   "sk-explicit",
		Endpoint: "https://example-openai-compatible.test/v1",
		ProxyURL: "",
		HTTPClientFactory: func(baseURL, apiKey, provider string) hermes.Client {
			httpCalls++
			seen = fakeHTTPClient{baseURL: baseURL, apiKey: apiKey, provider: provider}
			return fakeHTTPClient{baseURL: baseURL, apiKey: apiKey, provider: provider}
		},
		NativeClientFactory: func(_ runtime.NativeClientRequest) (hermes.Client, error) {
			nativeCalls++
			return fakeNativeClient{}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveBinding: %v", err)
	}

	if nativeCalls != 0 {
		t.Errorf("NativeClientFactory called %d times; want 0 when explicit endpoint set", nativeCalls)
	}
	if httpCalls != 1 {
		t.Fatalf("HTTPClientFactory called %d times; want 1", httpCalls)
	}
	if binding.EndpointSource != runtime.EndpointSourceExplicitEndpoint {
		t.Errorf("EndpointSource = %q; want %q", binding.EndpointSource, runtime.EndpointSourceExplicitEndpoint)
	}
	if binding.Endpoint != "https://example-openai-compatible.test/v1" {
		t.Errorf("Endpoint = %q; want explicit endpoint preserved", binding.Endpoint)
	}
	if seen.baseURL != "https://example-openai-compatible.test/v1" || seen.provider != "openai" {
		t.Errorf("HTTP factory request mismatch: %+v", seen)
	}
}

// TestResolveNativeRuntimeBinding_ProxyURLPreservesOpenAICompatible reuses the
// explicit-endpoint contract for the gateway proxy URL form.
func TestResolveNativeRuntimeBinding_ProxyURLPreservesOpenAICompatible(t *testing.T) {
	t.Parallel()

	httpCalls := 0
	binding, err := runtime.ResolveBinding(runtime.BindingRequest{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		APIKey:   "sk-explicit",
		Endpoint: "",
		ProxyURL: "https://gormes-proxy.example.test",
		HTTPClientFactory: func(baseURL, apiKey, provider string) hermes.Client {
			httpCalls++
			return fakeHTTPClient{baseURL: baseURL, apiKey: apiKey, provider: provider}
		},
		NativeClientFactory: func(runtime.NativeClientRequest) (hermes.Client, error) {
			t.Fatalf("native factory must not be invoked when proxy URL is set")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveBinding: %v", err)
	}

	if httpCalls != 1 {
		t.Errorf("HTTPClientFactory calls = %d; want 1", httpCalls)
	}
	if binding.EndpointSource != runtime.EndpointSourceExplicitEndpoint {
		t.Errorf("EndpointSource = %q; want %q for proxy URL", binding.EndpointSource, runtime.EndpointSourceExplicitEndpoint)
	}
	if binding.Endpoint != "https://gormes-proxy.example.test" {
		t.Errorf("Endpoint = %q; want proxy URL preserved", binding.Endpoint)
	}
}

// TestResolveNativeRuntimeBinding_ChannelNeutralFactoryUsedAcrossPlatforms
// covers acceptance bullet 3: the binding is exercised through a shared
// gateway/runtime factory used by channel adapters, preserving Telegram /
// Slack / Discord / WhatsApp / BlueBubbles / iMessage parity. The resolver
// must produce identical bindings regardless of the channel that invokes it.
func TestResolveNativeRuntimeBinding_ChannelNeutralFactoryUsedAcrossPlatforms(t *testing.T) {
	t.Parallel()

	channels := []string{"telegram", "slack", "discord", "whatsapp", "bluebubbles", "imessage"}
	var first runtime.Binding
	for i, channel := range channels {
		req := runtime.BindingRequest{
			Provider: "anthropic",
			Model:    "claude-sonnet-latest",
			APIKey:   "key",
			Endpoint: "",
			ProxyURL: "",
			Channel:  channel,
			HTTPClientFactory: func(string, string, string) hermes.Client {
				t.Fatalf("channel %q: HTTP factory must not run when no explicit endpoint", channel)
				return nil
			},
			NativeClientFactory: func(runtime.NativeClientRequest) (hermes.Client, error) {
				return fakeNativeClient{}, nil
			},
		}
		got, err := runtime.ResolveBinding(req)
		if err != nil {
			t.Fatalf("channel %q: ResolveBinding: %v", channel, err)
		}
		if i == 0 {
			first = got
			continue
		}
		if got.EndpointSource != first.EndpointSource {
			t.Errorf("channel %q: EndpointSource = %q; want %q (channel-neutral)",
				channel, got.EndpointSource, first.EndpointSource)
		}
		if got.Provider != first.Provider || got.Model != first.Model {
			t.Errorf("channel %q: provider/model drift: got=%+v want=%+v", channel, got, first)
		}
	}
}

// TestResolveNativeRuntimeBinding_DegradedWhenProviderConfigMissing covers the
// degraded_mode contract: incomplete native provider credentials/model
// routing must surface provider_config_missing/native_runtime_unavailable
// evidence WITHOUT silently dialling 127.0.0.1:8642 and WITHOUT importing
// hermes-agent runtime services.
func TestResolveNativeRuntimeBinding_DegradedWhenProviderConfigMissing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		req       runtime.BindingRequest
		wantCodes []string
	}{
		{
			name: "missing provider with no endpoint",
			req: runtime.BindingRequest{
				Provider: "",
				Model:    "claude-sonnet-latest",
				APIKey:   "key",
			},
			wantCodes: []string{runtime.DegradedReasonProviderConfigMissing, runtime.DegradedReasonNativeRuntimeUnavailable},
		},
		{
			name: "missing model with no endpoint",
			req: runtime.BindingRequest{
				Provider: "anthropic",
				Model:    "",
				APIKey:   "key",
			},
			wantCodes: []string{runtime.DegradedReasonProviderConfigMissing, runtime.DegradedReasonNativeRuntimeUnavailable},
		},
		{
			name: "missing api key with no endpoint",
			req: runtime.BindingRequest{
				Provider: "anthropic",
				Model:    "claude-sonnet-latest",
				APIKey:   "",
			},
			wantCodes: []string{runtime.DegradedReasonProviderConfigMissing, runtime.DegradedReasonNativeRuntimeUnavailable},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			httpCalls := 0
			nativeCalls := 0
			req.HTTPClientFactory = func(string, string, string) hermes.Client {
				httpCalls++
				return fakeHTTPClient{}
			}
			req.NativeClientFactory = func(runtime.NativeClientRequest) (hermes.Client, error) {
				nativeCalls++
				return fakeNativeClient{}, nil
			}

			binding, err := runtime.ResolveBinding(req)
			if err != nil {
				t.Fatalf("ResolveBinding: %v", err)
			}
			if httpCalls != 0 {
				t.Errorf("HTTP factory called %d times; degraded mode must not silently dial localhost", httpCalls)
			}
			if nativeCalls != 0 {
				t.Errorf("native factory called %d times; degraded mode must not build a half-configured runtime", nativeCalls)
			}
			if binding.EndpointSource != runtime.EndpointSourceUnconfigured {
				t.Errorf("EndpointSource = %q; want %q", binding.EndpointSource, runtime.EndpointSourceUnconfigured)
			}
			for _, want := range tc.wantCodes {
				if !containsString(binding.DegradedReasons, want) {
					t.Errorf("DegradedReasons = %v; want code %q", binding.DegradedReasons, want)
				}
			}
			if binding.Client != nil {
				t.Errorf("Client must be nil in degraded mode; got %T", binding.Client)
			}
			if strings.Contains(strings.Join(binding.DegradedReasons, " "), "127.0.0.1:8642") {
				t.Errorf("degraded reason mentions implicit localhost backend; must remain channel-neutral: %v", binding.DegradedReasons)
			}
		})
	}
}

// TestRuntimeStatusEvidence_RedactsSecrets covers acceptance bullet 4:
// redacted runtime status names provider, model, endpoint_source, and
// degraded reason without exposing secrets (api keys, proxy keys, tokens).
func TestRuntimeStatusEvidence_RedactsSecrets(t *testing.T) {
	t.Parallel()

	binding := runtime.Binding{
		Provider:        "anthropic",
		Model:           "claude-sonnet-latest",
		APIKey:          "sk-very-secret-anthropic",
		Endpoint:        "https://api.anthropic.com/v1",
		EndpointSource:  runtime.EndpointSourceExplicitEndpoint,
		DegradedReasons: []string{},
	}

	status := runtime.StatusEvidence(binding)

	for _, name := range []string{"provider", "model", "endpoint_source"} {
		if _, ok := status[name]; !ok {
			t.Errorf("status missing required field %q: %v", name, status)
		}
	}

	flat := flattenStatus(status)
	for _, secret := range []string{"sk-very-secret-anthropic", "very-secret"} {
		if strings.Contains(flat, secret) {
			t.Errorf("status leaks secret %q: %s", secret, flat)
		}
	}

	if got, _ := status["provider"].(string); got != "anthropic" {
		t.Errorf("status[provider] = %v; want anthropic", status["provider"])
	}
	if got, _ := status["model"].(string); got != "claude-sonnet-latest" {
		t.Errorf("status[model] = %v; want claude-sonnet-latest", status["model"])
	}
	if got, _ := status["endpoint_source"].(string); got != string(runtime.EndpointSourceExplicitEndpoint) {
		t.Errorf("status[endpoint_source] = %v; want %s", status["endpoint_source"], runtime.EndpointSourceExplicitEndpoint)
	}

	degradedBinding := runtime.Binding{
		Provider:        "",
		Model:           "claude-sonnet-latest",
		APIKey:          "sk-leak-bait",
		EndpointSource:  runtime.EndpointSourceUnconfigured,
		DegradedReasons: []string{runtime.DegradedReasonProviderConfigMissing, runtime.DegradedReasonNativeRuntimeUnavailable},
	}
	degradedStatus := runtime.StatusEvidence(degradedBinding)
	reasons, ok := degradedStatus["degraded_reasons"].([]string)
	if !ok {
		t.Fatalf("degraded status[degraded_reasons] missing or wrong type: %v", degradedStatus["degraded_reasons"])
	}
	if !containsString(reasons, runtime.DegradedReasonProviderConfigMissing) {
		t.Errorf("degraded_reasons missing %q: %v", runtime.DegradedReasonProviderConfigMissing, reasons)
	}
	flatDegraded := flattenStatus(degradedStatus)
	if strings.Contains(flatDegraded, "sk-leak-bait") {
		t.Errorf("degraded status leaks api key: %s", flatDegraded)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func flattenStatus(status map[string]any) string {
	var b strings.Builder
	for k, v := range status {
		b.WriteString(k)
		b.WriteByte('=')
		switch v := v.(type) {
		case string:
			b.WriteString(v)
		case []string:
			for _, s := range v {
				b.WriteString(s)
				b.WriteByte(',')
			}
		}
		b.WriteByte(' ')
	}
	return b.String()
}
