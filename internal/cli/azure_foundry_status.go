package cli

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// AzureFoundryManualOptions describes the manual fallback choices the
// CLI status should keep visible when probe-driven auto-detection cannot
// classify an endpoint. The options are rendered as a guidance block;
// they do not trigger any I/O or persistence themselves.
type AzureFoundryManualOptions struct {
	// APIModes is the list of api_mode values an operator may pick when
	// the probe yields manual_required. Empty means the CLI should fall
	// back to the canonical pair (openai_chat_completions,
	// anthropic_messages) so manual entry is always viable.
	APIModes []hermes.AzureTransport
	// ModelHint is an example deployment / model string surfaced in the
	// manual-entry guidance line. Empty values render a generic hint.
	ModelHint string
}

// AzureFoundryStatusInput is the deterministic input for rendering the
// CLI status surface. Both fields are pre-resolved read models - the
// renderer never contacts Azure, never reads files, and never prompts.
type AzureFoundryStatusInput struct {
	Runtime       hermes.AzureFoundryRuntime
	Probe         hermes.AzureProbeResult
	ManualOptions AzureFoundryManualOptions
}

// AzureFoundryStatus is the rendered output. Lines are the operator-
// facing rows; Evidence is a structured trail (a superset of the
// runtime's evidence) that downstream tooling consumes for parity with
// the upstream wizard's audit prints. ManualEntryAvailable is true when
// the operator can still configure api_mode + model by hand - the
// degraded_mode contract requires this to stay true whenever a probe
// or credential is missing so the CLI never strands the operator.
type AzureFoundryStatus struct {
	Lines                []string
	Evidence             []string
	ManualEntryAvailable bool
}

// Evidence keys the CLI status emits in addition to the runtime's own
// trail. They are referenced by the degraded_mode contract.
const (
	azureStatusEvidenceManualRequired = "azure_detect_manual_required"
	azureStatusEvidenceModelsFailed   = "azure_models_probe_failed"
)

// RenderAzureFoundryStatus turns a runtime + probe read model into a
// deterministic CLI status surface. Acceptance criteria covered:
//
//  1. Detected OpenAI-style, detected Anthropic-style, and manual-
//     required states all render distinct human-readable banners.
//  2. Manual api_mode and deployment / model entry stay available
//     whenever the runtime is degraded or the probe failed - even on
//     the zero-value input.
//  3. The redacted runtime evidence (azure_secret_redacted) and key
//     source label (env / config) flow through the rendered status so
//     operators can audit which layer supplied the credential.
//  4. The renderer is pure: it never opens browser auth, never reads
//     env directly, never performs HTTP, and never prompts.
func RenderAzureFoundryStatus(in AzureFoundryStatusInput) AzureFoundryStatus {
	out := AzureFoundryStatus{
		Lines:    make([]string, 0, 6),
		Evidence: make([]string, 0, len(in.Runtime.Evidence)+4),
	}

	// ── Detection banner ────────────────────────────────────────────
	manualRequired := isManualRequired(in)
	switch {
	case manualRequired:
		out.Lines = append(out.Lines, "Manual entry required: probe could not classify endpoint")
	case in.Runtime.APIMode == hermes.AzureTransportOpenAI:
		out.Lines = append(out.Lines, "Detected: OpenAI-style (POST /v1/chat/completions)")
	case in.Runtime.APIMode == hermes.AzureTransportCodexResponses:
		out.Lines = append(out.Lines, "Detected: OpenAI-style (POST /v1/responses)")
	case in.Runtime.APIMode == hermes.AzureTransportAnthropic:
		out.Lines = append(out.Lines, "Detected: Anthropic-style (POST /v1/messages)")
	default:
		out.Lines = append(out.Lines, "Manual entry required: probe could not classify endpoint")
	}

	// ── Endpoint / model rows ──────────────────────────────────────
	if in.Runtime.BaseURL != "" {
		out.Lines = append(out.Lines, fmt.Sprintf("Endpoint: %s (source=%s)",
			in.Runtime.BaseURL, in.Runtime.BaseURLSource))
	} else {
		out.Lines = append(out.Lines, "Endpoint: (unset - manual entry required)")
	}
	if in.Runtime.Model != "" {
		out.Lines = append(out.Lines, fmt.Sprintf("Model: %s", in.Runtime.Model))
	}

	// ── Credential row (redaction-aware) ───────────────────────────
	if in.Runtime.KeyAvailable {
		out.Lines = append(out.Lines, fmt.Sprintf(
			"API key: %s (source=%s, env=AZURE_FOUNDRY_API_KEY)",
			in.Runtime.KeyFingerprint, in.Runtime.KeySource))
	} else {
		out.Lines = append(out.Lines, "API key: (missing - set AZURE_FOUNDRY_API_KEY or enter manually)")
	}

	// ── Manual fallback guidance ───────────────────────────────────
	if manualRequired {
		modes := manualAPIModes(in.ManualOptions.APIModes)
		modeLabels := make([]string, 0, len(modes))
		for _, m := range modes {
			modeLabels = append(modeLabels, string(m))
		}
		out.Lines = append(out.Lines, fmt.Sprintf(
			"Manual api_mode options: %s", strings.Join(modeLabels, ", ")))
		hint := strings.TrimSpace(in.ManualOptions.ModelHint)
		if hint == "" {
			hint = "e.g. gpt-5.4, claude-sonnet-4-6"
		}
		out.Lines = append(out.Lines, fmt.Sprintf(
			"Manual model entry: type a deployment name (%s)", hint))
	}

	// ── Evidence assembly ──────────────────────────────────────────
	out.Evidence = append(out.Evidence, in.Runtime.Evidence...)
	if manualRequired {
		out.Evidence = append(out.Evidence, azureStatusEvidenceManualRequired)
	}
	if modelsProbeFailed(in.Probe) {
		out.Evidence = append(out.Evidence, azureStatusEvidenceModelsFailed)
	}

	out.ManualEntryAvailable = manualRequired || in.Runtime.Degraded
	return out
}

// isManualRequired collapses the runtime + probe signal into a single
// boolean: manual entry is required whenever the probe explicitly asks
// for it OR the runtime carries no api_mode at all (so the request
// builder cannot decide on its own).
func isManualRequired(in AzureFoundryStatusInput) bool {
	if strings.EqualFold(in.Probe.Reason, "manual_required") {
		return true
	}
	mode := in.Runtime.APIMode
	if mode == "" || mode == hermes.AzureTransportUnknown {
		return true
	}
	return false
}

// modelsProbeFailed reports whether the /models OpenAI shape probe was
// attempted but did not classify. Anthropic-only matches still count as
// a /models failure because the OpenAI catalog probe ran first; this
// keeps the evidence aligned with the upstream wizard's "models probe
// failed" diagnostic.
func modelsProbeFailed(probe hermes.AzureProbeResult) bool {
	if probe.Transport == hermes.AzureTransportOpenAI {
		return false
	}
	for _, line := range probe.Evidence {
		switch {
		case strings.Contains(line, "GET ") && strings.Contains(line, "/models"),
			strings.HasPrefix(line, "models_probe_"):
			return true
		}
	}
	return false
}

// manualAPIModes returns the list of api_mode options the CLI should
// offer for manual fallback. An empty configured list collapses to the
// canonical pair so manual entry is always viable - the degraded_mode
// contract requires "manual api_mode entry" to stay available even when
// no upstream config has been written.
func manualAPIModes(configured []hermes.AzureTransport) []hermes.AzureTransport {
	if len(configured) > 0 {
		return configured
	}
	return []hermes.AzureTransport{
		hermes.AzureTransportOpenAI,
		hermes.AzureTransportAnthropic,
	}
}
