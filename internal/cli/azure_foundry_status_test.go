package cli

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// runtimeWithKey is a tiny helper that builds a deterministic runtime
// read model carrying a redacted env-sourced API key. The status fixtures
// reuse it so each test can focus on the variant under inspection
// (probe transport, manual fallback, context known/unknown) without
// re-asserting the unchanged credential-resolution path.
func runtimeWithKey(probe hermes.AzureProbeResult, model string) hermes.AzureFoundryRuntime {
	return hermes.ResolveAzureFoundryRuntime(hermes.AzureFoundryRuntimeInput{
		Config: hermes.AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   model,
		},
		Env: func(name string) (string, bool) {
			if name == "AZURE_FOUNDRY_API_KEY" {
				return "azfk-secret-1234567890abcdef", true
			}
			return "", false
		},
		Probe: probe,
	})
}

func joinStatusLines(s AzureFoundryStatus) string {
	return strings.Join(append(append([]string{}, s.Lines...), s.Evidence...), " | ")
}

// TestAzureFoundryStatus_DetectedOpenAIStyle covers acceptance #1
// (detected OpenAI-style render) and the contract that probe evidence
// flows into the operator-visible status without a live setup wizard.
func TestAzureFoundryStatus_DetectedOpenAIStyle(t *testing.T) {
	probe := hermes.AzureProbeResult{
		Transport: hermes.AzureTransportOpenAI,
		Models:    []string{"gpt-5.4", "gpt-5.4-mini"},
		Reason:    "GET /models returned 2 model(s) - OpenAI-style endpoint",
		Evidence:  []string{"GET /models -> 200", "model_id=gpt-5.4", "model_id=gpt-5.4-mini"},
	}

	got := RenderAzureFoundryStatus(AzureFoundryStatusInput{
		Runtime: runtimeWithKey(probe, "gpt-5.4"),
		Probe:   probe,
	})

	joined := joinStatusLines(got)
	if !strings.Contains(joined, "Detected: OpenAI-style") {
		t.Fatalf("status missing OpenAI-style label: %s", joined)
	}
	if !strings.Contains(joined, "azure_api_mode=openai_chat_completions") {
		t.Fatalf("evidence missing api_mode record: %s", joined)
	}
	if !strings.Contains(joined, "model_id=gpt-5.4") {
		t.Fatalf("evidence missing probe model_id record: %s", joined)
	}
	if strings.Contains(joined, "azure_detect_manual_required") {
		t.Fatalf("status must not flag manual fallback when probe classified: %s", joined)
	}
	if strings.Contains(joined, "azure_models_probe_failed") {
		t.Fatalf("status must not flag models probe failure when /models classified: %s", joined)
	}
	if got.ManualEntryAvailable {
		t.Fatalf("ManualEntryAvailable = true, want false on detected OpenAI-style state")
	}
}

// TestAzureFoundryStatus_DetectedAnthropicStyle covers acceptance #1
// (detected Anthropic-style render). The Anthropic transport must
// surface as a distinct label so operators can tell the two endpoint
// shapes apart from the rendered status alone.
func TestAzureFoundryStatus_DetectedAnthropicStyle(t *testing.T) {
	probe := hermes.AzureProbeResult{
		Transport: hermes.AzureTransportAnthropic,
		Reason:    "POST /v1/messages returned 400 with Anthropic-shaped error",
		Evidence:  []string{"POST /v1/messages -> 400", "anthropic_probe_shape_match"},
	}

	got := RenderAzureFoundryStatus(AzureFoundryStatusInput{
		Runtime: runtimeWithKey(probe, "claude-sonnet-4-6"),
		Probe:   probe,
	})

	joined := joinStatusLines(got)
	if !strings.Contains(joined, "Detected: Anthropic-style") {
		t.Fatalf("status missing Anthropic-style label: %s", joined)
	}
	if strings.Contains(joined, "Detected: OpenAI-style") {
		t.Fatalf("status must not also claim OpenAI-style when Anthropic detected: %s", joined)
	}
	if !strings.Contains(joined, "azure_api_mode=anthropic_messages") {
		t.Fatalf("evidence missing api_mode record: %s", joined)
	}
	if got.ManualEntryAvailable {
		t.Fatalf("ManualEntryAvailable = true, want false on detected Anthropic-style state")
	}
}

// TestAzureFoundryStatus_ManualRequiredFallback covers acceptance #1
// (manual-required render) and acceptance #2 (manual api_mode and
// deployment/model entry stay available when every probe fails). The
// degraded_mode contract requires azure_detect_manual_required and
// azure_models_probe_failed evidence in this state.
func TestAzureFoundryStatus_ManualRequiredFallback(t *testing.T) {
	probe := hermes.AzureProbeResult{
		Transport: hermes.AzureTransportUnknown,
		Reason:    "manual_required",
		Evidence: []string{
			"GET https://res.openai.azure.com/openai/v1/models -> 403",
			"POST https://res.openai.azure.com/openai/v1/messages -> 401",
			"anthropic_probe_shape_mismatch",
		},
	}

	got := RenderAzureFoundryStatus(AzureFoundryStatusInput{
		Runtime: runtimeWithKey(probe, ""),
		Probe:   probe,
		ManualOptions: AzureFoundryManualOptions{
			APIModes: []hermes.AzureTransport{
				hermes.AzureTransportOpenAI,
				hermes.AzureTransportAnthropic,
			},
			ModelHint: "e.g. gpt-5.4, claude-sonnet-4-6",
		},
	})

	joined := joinStatusLines(got)
	if !strings.Contains(joined, "Manual entry required") {
		t.Fatalf("status missing manual-entry banner: %s", joined)
	}
	if !strings.Contains(joined, "azure_detect_manual_required") {
		t.Fatalf("evidence missing azure_detect_manual_required: %s", joined)
	}
	if !strings.Contains(joined, "azure_models_probe_failed") {
		t.Fatalf("evidence missing azure_models_probe_failed: %s", joined)
	}
	if !strings.Contains(joined, "openai_chat_completions") || !strings.Contains(joined, "anthropic_messages") {
		t.Fatalf("status must enumerate manual api_mode options for the operator: %s", joined)
	}
	if !strings.Contains(joined, "e.g. gpt-5.4, claude-sonnet-4-6") {
		t.Fatalf("status must surface manual model hint for operator entry: %s", joined)
	}
	if !got.ManualEntryAvailable {
		t.Fatalf("ManualEntryAvailable = false, want true when probe yields manual_required")
	}
}

// TestAzureFoundryStatus_RedactsAPIKeyAndNamesSource covers acceptance #3:
// the status must redact AZURE_FOUNDRY_API_KEY and identify the source
// label so operators can tell which env/config layer supplied the key.
func TestAzureFoundryStatus_RedactsAPIKeyAndNamesSource(t *testing.T) {
	const secret = "azfk-supersecret-doNOTleak-7890"
	runtime := hermes.ResolveAzureFoundryRuntime(hermes.AzureFoundryRuntimeInput{
		Config: hermes.AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "gpt-5.4",
		},
		Env: func(name string) (string, bool) {
			if name == "AZURE_FOUNDRY_API_KEY" {
				return secret, true
			}
			return "", false
		},
		Probe: hermes.AzureProbeResult{Transport: hermes.AzureTransportOpenAI},
	})

	got := RenderAzureFoundryStatus(AzureFoundryStatusInput{
		Runtime: runtime,
		Probe:   hermes.AzureProbeResult{Transport: hermes.AzureTransportOpenAI},
	})

	joined := joinStatusLines(got)
	if strings.Contains(joined, secret) {
		t.Fatalf("status leaks API key plaintext: %s", joined)
	}
	if !strings.Contains(joined, "azure_secret_redacted") {
		t.Fatalf("evidence missing azure_secret_redacted record: %s", joined)
	}
	if !strings.Contains(joined, "source=env") {
		t.Fatalf("status must name env as the key source: %s", joined)
	}
	if !strings.Contains(joined, "AZURE_FOUNDRY_API_KEY") {
		t.Fatalf("status must name AZURE_FOUNDRY_API_KEY as the env var consulted: %s", joined)
	}
}

// TestAzureFoundryStatus_MissingKeyKeepsManualEntry covers acceptance #2
// at the credential layer: when no API key is available the rendered
// status must still expose manual entry rather than declaring success.
// It also asserts that no spurious azure_secret_redacted evidence is
// emitted when there is no key to redact.
func TestAzureFoundryStatus_MissingKeyKeepsManualEntry(t *testing.T) {
	runtime := hermes.ResolveAzureFoundryRuntime(hermes.AzureFoundryRuntimeInput{
		Config: hermes.AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "gpt-5.4",
		},
		Env: func(string) (string, bool) { return "", false },
	})

	got := RenderAzureFoundryStatus(AzureFoundryStatusInput{
		Runtime: runtime,
		Probe:   hermes.AzureProbeResult{Reason: "manual_required"},
	})

	joined := joinStatusLines(got)
	if !strings.Contains(joined, "azure_foundry_key_missing") {
		t.Fatalf("evidence missing azure_foundry_key_missing: %s", joined)
	}
	if strings.Contains(joined, "azure_secret_redacted") {
		t.Fatalf("evidence must not record redaction when no key is available: %s", joined)
	}
	if !got.ManualEntryAvailable {
		t.Fatalf("ManualEntryAvailable = false, want true when key is missing")
	}
}

// TestAzureFoundryStatus_ContextUnknownPropagated covers degraded_mode:
// the CLI status echoes azure_context_unknown when the runtime cannot
// resolve a context length so an operator can see the gap from the
// status line alone.
func TestAzureFoundryStatus_ContextUnknownPropagated(t *testing.T) {
	probe := hermes.AzureProbeResult{Transport: hermes.AzureTransportOpenAI}
	runtime := hermes.ResolveAzureFoundryRuntime(hermes.AzureFoundryRuntimeInput{
		Config: hermes.AzureFoundryConfig{
			BaseURL: "https://res.openai.azure.com/openai/v1",
			Model:   "private-deployment",
		},
		Env: func(name string) (string, bool) {
			if name == "AZURE_FOUNDRY_API_KEY" {
				return "azfk-key", true
			}
			return "", false
		},
		Probe: probe,
	})

	got := RenderAzureFoundryStatus(AzureFoundryStatusInput{
		Runtime: runtime,
		Probe:   probe,
	})

	joined := joinStatusLines(got)
	if !strings.Contains(joined, "azure_context_unknown") {
		t.Fatalf("evidence missing azure_context_unknown: %s", joined)
	}
}

// TestAzureFoundryStatus_NoLiveCallsOrPrompts covers acceptance #4:
// the renderer is a pure read model. It never opens browser auth, never
// performs HTTP, never reads from os.Stdin, and never panics on the
// zero-value input. Calling it twice must yield identical output.
func TestAzureFoundryStatus_NoLiveCallsOrPrompts(t *testing.T) {
	first := RenderAzureFoundryStatus(AzureFoundryStatusInput{})
	second := RenderAzureFoundryStatus(AzureFoundryStatusInput{})

	if joinStatusLines(first) != joinStatusLines(second) {
		t.Fatalf("renderer is not deterministic: %v vs %v", first, second)
	}
	// A bare zero input is the "nothing configured at all" state - the
	// CLI must keep manual entry available so the operator has a path
	// forward without a setup wizard.
	if !first.ManualEntryAvailable {
		t.Fatalf("ManualEntryAvailable = false on zero-value input, want true")
	}
}
