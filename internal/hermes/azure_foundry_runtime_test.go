package hermes

import (
	"errors"
	"strings"
	"testing"
)

// envFunc returns a func(string) (string, bool) that reads from a fixed
// fixture map. It lets the tests inject AZURE_FOUNDRY_* env values
// without touching real os.Setenv state, so the runtime read model can
// be exercised with deterministic precedence semantics.
func envFunc(fixture map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		if fixture == nil {
			return "", false
		}
		v, ok := fixture[name]
		return v, ok
	}
}

// joinEvidence is a small helper used in assertions so test failure
// messages show the full evidence list as a single string.
func joinEvidence(ev []string) string { return strings.Join(ev, " | ") }

// TestAzureFoundryRuntime_EnvOverridesEmptyConfig asserts that
// AZURE_FOUNDRY_BASE_URL and AZURE_FOUNDRY_API_KEY override empty
// config values, that the resolved base URL has its trailing slash
// stripped, and that the rendered status redacts the key material with
// azure_secret_redacted evidence (acceptance #1).
func TestAzureFoundryRuntime_EnvOverridesEmptyConfig(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{},
		Env: envFunc(map[string]string{
			"AZURE_FOUNDRY_BASE_URL": "https://my-resource.openai.azure.com/openai/v1/",
			"AZURE_FOUNDRY_API_KEY":  "azfk-secret-1234567890abcdef",
		}),
	})

	if got.BaseURL != "https://my-resource.openai.azure.com/openai/v1" {
		t.Fatalf("BaseURL = %q, want trimmed env value", got.BaseURL)
	}
	if got.BaseURLSource != AzureFoundryBaseURLSourceEnv {
		t.Fatalf("BaseURLSource = %q, want %q", got.BaseURLSource, AzureFoundryBaseURLSourceEnv)
	}
	if !got.KeyAvailable {
		t.Fatal("KeyAvailable = false, want true (env supplied a key)")
	}
	if got.KeySource != AzureFoundryKeySourceEnv {
		t.Fatalf("KeySource = %q, want %q", got.KeySource, AzureFoundryKeySourceEnv)
	}
	if got.Degraded {
		t.Fatalf("Degraded = true, want false; evidence = %v", got.Evidence)
	}

	joined := joinEvidence(got.Evidence)
	if strings.Contains(joined, "azfk-secret-1234567890abcdef") {
		t.Fatalf("Evidence leaks API key plaintext: %q", joined)
	}
	if !strings.Contains(joined, "azure_secret_redacted") {
		t.Fatalf("Evidence = %v, want azure_secret_redacted record", got.Evidence)
	}
	if strings.Contains(joined, "azure_foundry_base_url_missing") {
		t.Fatalf("Evidence = %v, want no base_url_missing when env supplied URL", got.Evidence)
	}
	if strings.Contains(joined, "azure_foundry_key_missing") {
		t.Fatalf("Evidence = %v, want no key_missing when env supplied key", got.Evidence)
	}
}

// TestAzureFoundryRuntime_RedactsKeyAndNeverStoresPlaintext asserts
// that the runtime read model never embeds the API key in any
// observable output - only KeyAvailable and a redacted fingerprint.
// The contract is "without ... storing plaintext secrets".
func TestAzureFoundryRuntime_RedactsKeyAndNeverStoresPlaintext(t *testing.T) {
	const secret = "azfk-supersecret-doNOTleak-7890"
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{BaseURL: "https://res.openai.azure.com/openai/v1"},
		Env: envFunc(map[string]string{
			"AZURE_FOUNDRY_API_KEY": secret,
		}),
	})

	for _, e := range got.Evidence {
		if strings.Contains(e, secret) {
			t.Fatalf("Evidence entry leaks secret: %q", e)
		}
	}
	if got.KeyFingerprint == secret {
		t.Fatalf("KeyFingerprint stores the plaintext key: %q", got.KeyFingerprint)
	}
	if got.KeyFingerprint == "" {
		t.Fatal("KeyFingerprint = empty when key is available; want a non-empty redacted preview")
	}
	if strings.Contains(got.KeyFingerprint, secret[6:]) {
		t.Fatalf("KeyFingerprint = %q exposes too much of the secret", got.KeyFingerprint)
	}
}

// TestAzureFoundryRuntime_ConfigBeatsEnvForBaseURL asserts that an
// explicit config base_url (the user already ran 'hermes model')
// takes precedence over AZURE_FOUNDRY_BASE_URL env. That mirrors the
// upstream _resolve_azure_foundry_runtime precedence: explicit >
// config > env.
func TestAzureFoundryRuntime_ConfigBeatsEnvForBaseURL(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://from-config.openai.azure.com/openai/v1",
		},
		Env: envFunc(map[string]string{
			"AZURE_FOUNDRY_BASE_URL": "https://from-env.openai.azure.com/openai/v1",
			"AZURE_FOUNDRY_API_KEY":  "key",
		}),
	})

	if got.BaseURL != "https://from-config.openai.azure.com/openai/v1" {
		t.Fatalf("BaseURL = %q, want config to win", got.BaseURL)
	}
	if got.BaseURLSource != AzureFoundryBaseURLSourceConfig {
		t.Fatalf("BaseURLSource = %q, want %q", got.BaseURLSource, AzureFoundryBaseURLSourceConfig)
	}
}

// TestAzureFoundryRuntime_StripsTrailingV1ForAnthropicMessages asserts
// that for anthropic_messages api_mode, a configured /v1 suffix is
// stripped from the base URL because the Anthropic SDK appends
// /v1/messages itself. Mirrors the cfg_api_mode == "anthropic_messages"
// branch of _resolve_azure_foundry_runtime.
func TestAzureFoundryRuntime_StripsTrailingV1ForAnthropicMessages(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.services.ai.azure.com/anthropic/v1",
			APIMode: AzureTransportAnthropic,
			Model:   "claude-sonnet-4-6",
		},
		Env: envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
	})

	if got.BaseURL != "https://res.services.ai.azure.com/anthropic" {
		t.Fatalf("BaseURL = %q, want trailing /v1 stripped for anthropic_messages", got.BaseURL)
	}
	if got.APIMode != AzureTransportAnthropic {
		t.Fatalf("APIMode = %q, want %q", got.APIMode, AzureTransportAnthropic)
	}
}

// TestAzureFoundryRuntime_CombinesProbeResultWithoutChangingBuilders
// asserts that a typed AzureProbeResult contributes api_mode and
// advisory evidence into the runtime read model when config does not
// already pin api_mode and model-family inference has no override. This is
// acceptance #2: probe + config compose without touching request-builder
// behavior.
func TestAzureFoundryRuntime_CombinesProbeResultWithoutChangingBuilders(t *testing.T) {
	probe := AzureProbeResult{
		Transport: AzureTransportOpenAI,
		Models:    []string{"gpt-4.1", "gpt-4o"},
		Reason:    "GET /models returned 2 model(s) - OpenAI-style endpoint",
		Evidence:  []string{"GET /models -> 200", "model_id=gpt-4.1", "model_id=gpt-4o"},
	}

	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "gpt-4.1",
		},
		Env:   envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
		Probe: probe,
	})

	if got.APIMode != AzureTransportOpenAI {
		t.Fatalf("APIMode = %q, want probe to supply openai_chat_completions", got.APIMode)
	}
	if got.APIModeSource != AzureFoundryAPIModeSourceProbe {
		t.Fatalf("APIModeSource = %q, want %q", got.APIModeSource, AzureFoundryAPIModeSourceProbe)
	}
	if got.Model != "gpt-4.1" {
		t.Fatalf("Model = %q, want config-pinned deployment", got.Model)
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_probe_transport=openai_chat_completions") {
		t.Fatalf("Evidence = %v, want a probe transport record", got.Evidence)
	}
	if !strings.Contains(joined, "model_id=gpt-4.1") {
		t.Fatalf("Evidence = %v, want probe model evidence forwarded", got.Evidence)
	}
}

// TestAzureFoundryRuntime_ConfigAPIModeBeatsProbe asserts that an
// explicit api_mode in config.yaml wins over a probe transport that
// disagrees - the operator's pinned choice must not be silently
// overridden by an auto-detected transport.
func TestAzureFoundryRuntime_ConfigAPIModeBeatsProbe(t *testing.T) {
	probe := AzureProbeResult{
		Transport: AzureTransportOpenAI,
		Reason:    "GET /models returned 1 model - OpenAI-style endpoint",
	}

	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			APIMode: AzureTransportAnthropic,
			Model:   "claude-sonnet-4-6",
		},
		Env:   envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
		Probe: probe,
	})

	if got.APIMode != AzureTransportAnthropic {
		t.Fatalf("APIMode = %q, want config-pinned anthropic_messages to win", got.APIMode)
	}
	if got.APIModeSource != AzureFoundryAPIModeSourceConfig {
		t.Fatalf("APIModeSource = %q, want %q", got.APIModeSource, AzureFoundryAPIModeSourceConfig)
	}
}

// TestAzureFoundryRuntime_KnownContextLengthRecorded asserts that when
// the model context resolver finds a known context length, the runtime
// emits azure_context_known evidence and exposes the resolved value.
// Acceptance #3.
func TestAzureFoundryRuntime_KnownContextLengthRecorded(t *testing.T) {
	caps := StaticModelContextCaps{
		ModelContextKey{Provider: "azure-foundry", Model: "gpt-5.4"}: 400_000,
	}

	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "gpt-5.4",
		},
		Env:           envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
		Probe:         AzureProbeResult{Transport: AzureTransportOpenAI},
		ContextLookup: caps,
	})

	if got.Context.ContextLength != 400_000 {
		t.Fatalf("Context.ContextLength = %d, want 400000 from caps", got.Context.ContextLength)
	}
	if got.Context.Source != ModelContextSourceProviderCap {
		t.Fatalf("Context.Source = %q, want %q", got.Context.Source, ModelContextSourceProviderCap)
	}
	if !got.Context.Known() {
		t.Fatalf("Context.Known() = false, want true")
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_context_known") {
		t.Fatalf("Evidence = %v, want azure_context_known record", got.Evidence)
	}
	if strings.Contains(joined, "azure_context_unknown") {
		t.Fatalf("Evidence = %v, must not record both known and unknown", got.Evidence)
	}
}

// TestAzureFoundryRuntime_UnknownContextLengthRecorded asserts that
// when the model context resolver finds no known length and there is
// no models.dev fallback, the runtime emits azure_context_unknown
// evidence. Acceptance #3 (negative half).
func TestAzureFoundryRuntime_UnknownContextLengthRecorded(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "private-deployment",
		},
		Env: envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
		// No ContextLookup, no ModelInfo: explicitly unknown.
	})

	if got.Context.Known() {
		t.Fatalf("Context.Known() = true, want false; resolution = %+v", got.Context)
	}
	if got.Context.ContextLength != 0 {
		t.Fatalf("Context.ContextLength = %d, want 0", got.Context.ContextLength)
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_context_unknown") {
		t.Fatalf("Evidence = %v, want azure_context_unknown record", got.Evidence)
	}
}

// TestAzureFoundryRuntime_ModelInfoFallbackStillRecordsKnownContext
// asserts that a models.dev-style fallback (no provider-cap entry but a
// non-zero ModelInfo.ContextWindow) counts as known context. The
// degraded_mode wording calls this out explicitly: provider context
// metadata participates alongside provider-cap data.
func TestAzureFoundryRuntime_ModelInfoFallbackStillRecordsKnownContext(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "fallback-model",
		},
		Env:       envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
		ModelInfo: ModelContextMetadata{ContextWindow: 200_000},
	})

	if got.Context.ContextLength != 200_000 {
		t.Fatalf("Context.ContextLength = %d, want 200000 from ModelInfo fallback", got.Context.ContextLength)
	}
	if got.Context.Source != ModelContextSourceModelsDev {
		t.Fatalf("Context.Source = %q, want %q", got.Context.Source, ModelContextSourceModelsDev)
	}
	if !strings.Contains(joinEvidence(got.Evidence), "azure_context_known") {
		t.Fatalf("Evidence = %v, want azure_context_known when ModelInfo supplies length", got.Evidence)
	}
}

// TestAzureFoundryRuntime_MissingBaseURLDegrades asserts that with no
// config and no env URL the runtime is degraded but not fatal. The
// degraded result keeps manual endpoint configuration possible
// (acceptance #4).
func TestAzureFoundryRuntime_MissingBaseURLDegrades(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{},
		Env:    envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
	})

	if !got.Degraded {
		t.Fatalf("Degraded = false, want true when base URL is missing")
	}
	if got.BaseURL != "" {
		t.Fatalf("BaseURL = %q, want empty when missing", got.BaseURL)
	}
	if got.BaseURLSource != AzureFoundryBaseURLSourceUnset {
		t.Fatalf("BaseURLSource = %q, want %q", got.BaseURLSource, AzureFoundryBaseURLSourceUnset)
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_foundry_base_url_missing") {
		t.Fatalf("Evidence = %v, want azure_foundry_base_url_missing", got.Evidence)
	}
}

// TestAzureFoundryRuntime_MissingAPIKeyDegrades asserts that a missing
// API key (no env, no explicit) emits azure_foundry_key_missing and
// flags Degraded so the wizard can keep manual entry available.
// Acceptance #4 (key half).
func TestAzureFoundryRuntime_MissingAPIKeyDegrades(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "gpt-5.4",
		},
		Env: envFunc(nil),
	})

	if !got.Degraded {
		t.Fatalf("Degraded = false, want true when key is missing")
	}
	if got.KeyAvailable {
		t.Fatalf("KeyAvailable = true, want false when key is missing")
	}
	if got.KeySource != AzureFoundryKeySourceUnset {
		t.Fatalf("KeySource = %q, want %q", got.KeySource, AzureFoundryKeySourceUnset)
	}
	if got.KeyFingerprint != "" {
		t.Fatalf("KeyFingerprint = %q, want empty when no key", got.KeyFingerprint)
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_foundry_key_missing") {
		t.Fatalf("Evidence = %v, want azure_foundry_key_missing", got.Evidence)
	}
	if strings.Contains(joined, "azure_secret_redacted") {
		t.Fatalf("Evidence = %v, must not record redaction when no key was provided", got.Evidence)
	}
}

// TestAzureFoundryRuntime_NoLiveAzureDependency asserts that the
// resolver can be exercised purely from fixtures - it never receives
// nor calls an HTTP client, and never returns a transport-style error.
// The done_signal explicitly demands "no live Azure dependency".
func TestAzureFoundryRuntime_NoLiveAzureDependency(t *testing.T) {
	// Inject a context lookup that errors. The resolver must record
	// the failure as evidence rather than panicking or escalating it
	// to a fatal error.
	failing := ModelContextLookupFunc(func(ModelContextQuery) (int, bool, error) {
		return 0, false, errors.New("registry unavailable")
	})

	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "gpt-5.4",
		},
		Env:           envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
		ContextLookup: failing,
	})

	if got.Context.ProviderLookupError == "" {
		t.Fatal("Context.ProviderLookupError = empty, want the failing-lookup error to be recorded")
	}
	if got.Context.Known() {
		t.Fatalf("Context.Known() = true, want false on lookup error without ModelInfo")
	}
	if !strings.Contains(joinEvidence(got.Evidence), "azure_context_unknown") {
		t.Fatalf("Evidence = %v, want azure_context_unknown when lookup fails", got.Evidence)
	}
}

// TestAzureFoundryRuntime_DefaultLookupEnvIsOSEnv asserts that when no
// Env function is supplied, the resolver still reads AZURE_FOUNDRY_*
// from the process environment. The fixture installs sentinel values
// via t.Setenv which the resolver must observe.
func TestAzureFoundryRuntime_DefaultLookupEnvIsOSEnv(t *testing.T) {
	t.Setenv("AZURE_FOUNDRY_BASE_URL", "https://from-osenv.openai.azure.com/openai/v1")
	t.Setenv("AZURE_FOUNDRY_API_KEY", "osenv-key")

	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		// Env left nil - resolver must default to os.LookupEnv.
	})

	if got.BaseURL != "https://from-osenv.openai.azure.com/openai/v1" {
		t.Fatalf("BaseURL = %q, want value from os env", got.BaseURL)
	}
	if !got.KeyAvailable {
		t.Fatal("KeyAvailable = false, want true (os env supplied a key)")
	}
}

func TestAzureFoundryRuntime_TargetModelOverridesDefault(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			APIMode: AzureTransportOpenAI,
			Model:   "gpt-4o",
		},
		TargetModel: "gpt-5.3-codex",
		Env:         envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
	})

	if got.Model != "gpt-5.3-codex" {
		t.Fatalf("Model = %q, want target_model to override stale default", got.Model)
	}
	if got.APIMode != AzureTransportCodexResponses {
		t.Fatalf("APIMode = %q, want %q for target_model", got.APIMode, AzureTransportCodexResponses)
	}
	if got.APIModeSource != AzureFoundryAPIModeSourceInferred {
		t.Fatalf("APIModeSource = %q, want %q", got.APIModeSource, AzureFoundryAPIModeSourceInferred)
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_foundry_api_mode_inferred") {
		t.Fatalf("Evidence = %v, want azure_foundry_api_mode_inferred", got.Evidence)
	}
	if strings.Contains(joined, "azure_foundry_api_mode_unknown") {
		t.Fatalf("Evidence = %v, must not report unknown after target_model inference", got.Evidence)
	}
}

func TestAzureFoundryRuntime_AnthropicMessagesGuard(t *testing.T) {
	got := ResolveAzureFoundryRuntime(AzureFoundryRuntimeInput{
		Config: AzureFoundryConfig{
			BaseURL: "https://res.services.ai.azure.com/anthropic/v1",
			APIMode: AzureTransportAnthropic,
			Model:   "gpt-5.3-codex",
		},
		Env: envFunc(map[string]string{"AZURE_FOUNDRY_API_KEY": "key"}),
	})

	if got.APIMode != AzureTransportAnthropic {
		t.Fatalf("APIMode = %q, want explicit %q preserved", got.APIMode, AzureTransportAnthropic)
	}
	if got.APIModeSource != AzureFoundryAPIModeSourceConfig {
		t.Fatalf("APIModeSource = %q, want %q", got.APIModeSource, AzureFoundryAPIModeSourceConfig)
	}
	if got.BaseURL != "https://res.services.ai.azure.com/anthropic" {
		t.Fatalf("BaseURL = %q, want trailing /v1 stripped for preserved anthropic_messages", got.BaseURL)
	}
	joined := joinEvidence(got.Evidence)
	if !strings.Contains(joined, "azure_foundry_api_mode_preserved") {
		t.Fatalf("Evidence = %v, want azure_foundry_api_mode_preserved", got.Evidence)
	}
	if strings.Contains(joined, "azure_foundry_api_mode_inferred") {
		t.Fatalf("Evidence = %v, must not infer over explicit anthropic_messages", got.Evidence)
	}
}
