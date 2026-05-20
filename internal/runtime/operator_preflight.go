package runtime

import (
	"strings"
)

const (
	PreflightReasonMissingProvider    = "provider_missing"
	PreflightReasonMissingModel       = "model_missing"
	PreflightReasonMissingCredential  = "credential_missing"
	PreflightReasonMissingEndpoint    = "endpoint_missing"
	PreflightReasonNativeUnavailable  = "native_runtime_unavailable"
)

// OperatorPreflightInput carries the already-resolved provider/model/credential
// values into the preflight checker. It is pure — no network clients, no config
// mutation, no OAuth flows.
type OperatorPreflightInput struct {
	Provider string
	Model    string
	APIKey   string
	Endpoint string
}

// OperatorPreflightResult is the output of RunOperatorPreflight. It contains
// stable degraded reason codes and recommended commands suitable for feeding
// into OperatorRunReport without exposing secrets.
type OperatorPreflightResult struct {
	Ready             bool
	DegradedReason    string
	RecommendedCommand string
	input             OperatorPreflightInput
}

// Evidence returns a redacted status map suitable for OperatorRunReport
// runtime evidence. API keys are never exposed raw.
func (r OperatorPreflightResult) Evidence() map[string]any {
	ev := map[string]any{
		"provider": r.input.Provider,
		"model":    r.input.Model,
	}
	if r.input.APIKey != "" {
		ev["api_key"] = redactAPIKey(r.input.APIKey)
	}
	if r.input.Endpoint != "" {
		ev["endpoint"] = redactEndpoint(r.input.Endpoint)
	}
	if r.DegradedReason != "" {
		ev["degraded_reason"] = r.DegradedReason
	}
	return ev
}

func redactAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func redactEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	// Redact query parameters that may contain tokens
	if idx := strings.Index(endpoint, "?"); idx >= 0 {
		return endpoint[:idx] + "?[redacted]"
	}
	return endpoint
}

// RunOperatorPreflight performs a pure readiness check for unattended cron/fleet
// jobs. It validates that provider, model, and credential resolution is complete
// before execution would begin. No network calls, config mutation, or OAuth flows
// occur.
func RunOperatorPreflight(in OperatorPreflightInput) OperatorPreflightResult {
	provider := strings.TrimSpace(in.Provider)
	model := strings.TrimSpace(in.Model)
	apiKey := strings.TrimSpace(in.APIKey)
	endpoint := strings.TrimSpace(in.Endpoint)

	if provider == "" {
		return OperatorPreflightResult{
			Ready:              false,
			DegradedReason:     PreflightReasonMissingProvider,
			RecommendedCommand: "gormes setup provider",
			input:              in,
		}
	}

	if model == "" {
		return OperatorPreflightResult{
			Ready:              false,
			DegradedReason:     PreflightReasonMissingModel,
			RecommendedCommand: "gormes setup provider",
			input:              in,
		}
	}

	if apiKey == "" && endpoint == "" {
		return OperatorPreflightResult{
			Ready:              false,
			DegradedReason:     PreflightReasonMissingCredential,
			RecommendedCommand: "gormes auth status",
			input:              in,
		}
	}

	return OperatorPreflightResult{
		Ready: true,
		input: in,
	}
}
