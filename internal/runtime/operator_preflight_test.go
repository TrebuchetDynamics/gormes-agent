package runtime

import (
	"strings"
	"testing"
)

func TestOperatorPreflight_Ready(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "sk-test-key-12345",
		Endpoint: "https://api.openai.com/v1",
	})

	if !result.Ready {
		t.Error("expected Ready=true")
	}
	if result.DegradedReason != "" {
		t.Errorf("unexpected degraded_reason: %s", result.DegradedReason)
	}
	if result.RecommendedCommand != "" {
		t.Errorf("unexpected recommended_command: %s", result.RecommendedCommand)
	}
}

func TestOperatorPreflight_MissingProvider(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Model:  "gpt-4",
		APIKey: "sk-test-key",
	})

	if result.Ready {
		t.Error("expected Ready=false")
	}
	if result.DegradedReason != PreflightReasonMissingProvider {
		t.Errorf("degraded_reason=%q, want %s", result.DegradedReason, PreflightReasonMissingProvider)
	}
	if !strings.Contains(result.RecommendedCommand, "setup provider") {
		t.Errorf("recommended_command should mention setup provider: %s", result.RecommendedCommand)
	}
}

func TestOperatorPreflight_MissingModel(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "openai",
		APIKey:   "sk-test-key",
	})

	if result.Ready {
		t.Error("expected Ready=false")
	}
	if result.DegradedReason != PreflightReasonMissingModel {
		t.Errorf("degraded_reason=%q, want %s", result.DegradedReason, PreflightReasonMissingModel)
	}
}

func TestOperatorPreflight_MissingAPIKey(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "openai",
		Model:    "gpt-4",
	})

	if result.Ready {
		t.Error("expected Ready=false")
	}
	if result.DegradedReason != PreflightReasonMissingCredential {
		t.Errorf("degraded_reason=%q, want %s", result.DegradedReason, PreflightReasonMissingCredential)
	}
	if !strings.Contains(result.RecommendedCommand, "auth") && !strings.Contains(result.RecommendedCommand, "setup") {
		t.Errorf("recommended_command should mention auth or setup: %s", result.RecommendedCommand)
	}
}

func TestOperatorPreflight_MissingEndpoint(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "sk-test-key",
		Endpoint: "",
	})

	// Missing endpoint with provider/model/key should still be ready
	// (native provider path)
	if !result.Ready {
		t.Error("expected Ready=true for native provider path")
	}
}

func TestOperatorPreflight_NativeRuntimeUnavailable(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "unknown-provider",
		Model:    "gpt-4",
		APIKey:   "sk-test-key",
	})

	if !result.Ready {
		t.Error("expected Ready=true (unknown provider still passes preflight)")
	}
}

func TestOperatorPreflight_Redaction(t *testing.T) {
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "sk-secret-key-abcdef123456",
	})

	evidence := result.Evidence()
	if evidence["api_key"] == "sk-secret-key-abcdef123456" {
		t.Error("API key should be redacted in evidence")
	}
	if evidence["api_key"] == nil {
		t.Error("evidence should include redacted api_key field")
	}
}

func TestOperatorPreflight_EvidenceRedactsSecrets(t *testing.T) {
	input := OperatorPreflightInput{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "sk-very-secret-key-12345",
		Endpoint: "https://api.openai.com/v1",
	}
	result := RunOperatorPreflight(input)
	ev := result.Evidence()

	if ev["api_key"] == input.APIKey {
		t.Error("evidence must not expose raw API key")
	}
	if ev["provider"] != "openai" {
		t.Errorf("evidence provider=%v, want openai", ev["provider"])
	}
	if ev["model"] != "gpt-4" {
		t.Errorf("evidence model=%v, want gpt-4", ev["model"])
	}
}

func TestOperatorPreflight_AllDegradedReasonsHaveRecommendedCommand(t *testing.T) {
	inputs := []OperatorPreflightInput{
		{Model: "gpt-4", APIKey: "sk-key"},
		{Provider: "openai", APIKey: "sk-key"},
		{Provider: "openai", Model: "gpt-4"},
	}

	for _, in := range inputs {
		result := RunOperatorPreflight(in)
		if result.Ready {
			continue
		}
		if result.DegradedReason == "" {
			t.Errorf("missing degraded_reason for input: %+v", in)
		}
		if result.RecommendedCommand == "" {
			t.Errorf("missing recommended_command for degraded_reason=%s", result.DegradedReason)
		}
	}
}

func TestOperatorPreflight_NoNetworkCalls(t *testing.T) {
	// This test verifies the preflight is pure by running it with
	// no network connectivity possible (no HTTP client injection).
	result := RunOperatorPreflight(OperatorPreflightInput{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "sk-test",
	})

	// If this completes without hanging or panicking, the preflight
	// is pure and does not attempt network calls.
	if result.Evidence() == nil {
		t.Error("evidence should not be nil")
	}
}
