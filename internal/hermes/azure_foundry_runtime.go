package hermes

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// AzureFoundryConfig is the subset of config.yaml that drives Azure
// Foundry runtime resolution. Empty values mean "fall back to env".
type AzureFoundryConfig struct {
	// BaseURL is the configured endpoint (model.base_url in
	// config.yaml). Trailing slashes are stripped before precedence.
	BaseURL string
	// APIMode pins the request shape Azure Foundry expects. An empty
	// value means "let the probe decide". A non-empty value wins over
	// any probe transport so an operator's explicit choice is never
	// silently overridden.
	APIMode AzureTransport
	// Model is the configured deployment / model name.
	Model string
}

// AzureFoundryExplicit captures explicit overrides (e.g., flags or
// kwargs to a setup wizard call). Empty fields mean "no override".
type AzureFoundryExplicit struct {
	BaseURL string
	APIKey  string
}

// AzureFoundryBaseURLSource records which precedence layer supplied
// the resolved base URL. Used by status surfaces so operators can see
// whether the configured value or the env fallback won.
type AzureFoundryBaseURLSource string

const (
	AzureFoundryBaseURLSourceUnset    AzureFoundryBaseURLSource = "unset"
	AzureFoundryBaseURLSourceExplicit AzureFoundryBaseURLSource = "explicit"
	AzureFoundryBaseURLSourceConfig   AzureFoundryBaseURLSource = "config"
	AzureFoundryBaseURLSourceEnv      AzureFoundryBaseURLSource = "env"
)

// AzureFoundryKeySource records where the API key came from. The key
// itself never leaves this struct; only the source label and a short
// redacted fingerprint are exposed.
type AzureFoundryKeySource string

const (
	AzureFoundryKeySourceUnset    AzureFoundryKeySource = "unset"
	AzureFoundryKeySourceExplicit AzureFoundryKeySource = "explicit"
	AzureFoundryKeySourceEnv      AzureFoundryKeySource = "env"
)

// AzureFoundryAPIModeSource records whether api_mode was pinned by
// config or auto-detected by the typed probe. Lets status callers
// explain the chosen transport without re-deriving it.
type AzureFoundryAPIModeSource string

const (
	AzureFoundryAPIModeSourceUnset    AzureFoundryAPIModeSource = "unset"
	AzureFoundryAPIModeSourceConfig   AzureFoundryAPIModeSource = "config"
	AzureFoundryAPIModeSourceProbe    AzureFoundryAPIModeSource = "probe"
	AzureFoundryAPIModeSourceInferred AzureFoundryAPIModeSource = "inferred"
)

// AzureFoundryRuntimeInput is the deterministic fixture that drives
// ResolveAzureFoundryRuntime. The resolver never opens an HTTP client
// nor reads files - everything it sees is on this struct.
type AzureFoundryRuntimeInput struct {
	Config   AzureFoundryConfig
	Explicit AzureFoundryExplicit
	// TargetModel is the current request's model override. When present,
	// it wins over Config.Model for runtime mode inference and context lookup.
	TargetModel string
	// Env optionally overrides os.LookupEnv. Used by tests to inject
	// AZURE_FOUNDRY_* values without touching real process env.
	Env func(string) (string, bool)
	// Probe is the typed read model from the Azure Foundry transport
	// probe slice. The zero value is "no probe ran" and contributes
	// nothing to api_mode resolution.
	Probe AzureProbeResult
	// ContextLookup is the provider-cap registry used by the
	// model-context resolver. The zero value falls through to
	// ModelInfo and ultimately reports azure_context_unknown.
	ContextLookup ModelContextLookup
	// ModelInfo is the models.dev / vendor metadata fallback for
	// context length. Optional.
	ModelInfo ModelContextMetadata
}

// AzureFoundryRuntime is the redacted read-model the kernel and CLI
// status surfaces consume. It carries enough metadata to plan a
// request without ever holding plaintext credentials in observable
// state - the API key itself stays inside the resolver and only a
// fingerprint plus the KeyAvailable flag escape.
type AzureFoundryRuntime struct {
	Provider       string
	APIMode        AzureTransport
	APIModeSource  AzureFoundryAPIModeSource
	BaseURL        string
	BaseURLSource  AzureFoundryBaseURLSource
	Model          string
	KeyAvailable   bool
	KeySource      AzureFoundryKeySource
	KeyFingerprint string
	Context        ModelContextResolution
	Probe          AzureProbeResult
	Evidence       []string
	Degraded       bool
}

var azureFoundryAnthropicV1Suffix = regexp.MustCompile(`(?i)/v1/?$`)

// ResolveAzureFoundryRuntime computes the redacted Azure Foundry
// runtime read model from deterministic inputs. It never contacts
// Azure, never writes to disk, never mutates the process environment,
// and never embeds the API key in any observable field. Missing
// base_url or api_key flag the runtime as Degraded but never panic
// or return an error - the wizard must keep manual configuration
// possible (acceptance #4).
func ResolveAzureFoundryRuntime(in AzureFoundryRuntimeInput) AzureFoundryRuntime {
	lookup := in.Env
	if lookup == nil {
		lookup = os.LookupEnv
	}

	out := AzureFoundryRuntime{
		Provider:      "azure-foundry",
		APIModeSource: AzureFoundryAPIModeSourceUnset,
		BaseURLSource: AzureFoundryBaseURLSourceUnset,
		KeySource:     AzureFoundryKeySourceUnset,
		Model:         azureFoundryEffectiveModel(in.Config.Model, in.TargetModel),
		Probe:         in.Probe,
	}

	// ── Base URL precedence: explicit > config > env ───────────────
	explicitBase := normalizeAzureBaseURL(in.Explicit.BaseURL)
	configBase := normalizeAzureBaseURL(in.Config.BaseURL)
	envBase := normalizeAzureBaseURL(envValue(lookup, "AZURE_FOUNDRY_BASE_URL"))

	switch {
	case explicitBase != "":
		out.BaseURL = explicitBase
		out.BaseURLSource = AzureFoundryBaseURLSourceExplicit
	case configBase != "":
		out.BaseURL = configBase
		out.BaseURLSource = AzureFoundryBaseURLSourceConfig
	case envBase != "":
		out.BaseURL = envBase
		out.BaseURLSource = AzureFoundryBaseURLSourceEnv
	}

	// ── API mode: config wins over probe wins over unset ──────────
	switch {
	case in.Config.APIMode != "" && in.Config.APIMode != AzureTransportUnknown:
		out.APIMode = in.Config.APIMode
		out.APIModeSource = AzureFoundryAPIModeSourceConfig
	case in.Probe.Transport != "" && in.Probe.Transport != AzureTransportUnknown:
		out.APIMode = in.Probe.Transport
		out.APIModeSource = AzureFoundryAPIModeSourceProbe
	default:
		out.APIMode = AzureTransportUnknown
	}
	if out.APIMode != AzureTransportAnthropic {
		if inferred, ok := AzureFoundryAPIModeForModel(out.Model); ok {
			out.APIMode = inferred
			out.APIModeSource = AzureFoundryAPIModeSourceInferred
		}
	}

	// Anthropic SDK appends /v1/messages itself - strip a trailing
	// /v1 from the configured base URL to avoid double-/v1 paths.
	if out.APIMode == AzureTransportAnthropic && out.BaseURL != "" {
		out.BaseURL = azureFoundryAnthropicV1Suffix.ReplaceAllString(out.BaseURL, "")
	}

	// ── API key precedence: explicit > env ─────────────────────────
	explicitKey := strings.TrimSpace(in.Explicit.APIKey)
	envKey := strings.TrimSpace(envValue(lookup, "AZURE_FOUNDRY_API_KEY"))

	var resolvedKey string
	switch {
	case explicitKey != "":
		resolvedKey = explicitKey
		out.KeySource = AzureFoundryKeySourceExplicit
	case envKey != "":
		resolvedKey = envKey
		out.KeySource = AzureFoundryKeySourceEnv
	}
	if resolvedKey != "" {
		out.KeyAvailable = true
		out.KeyFingerprint = redactAzureFoundryKey(resolvedKey)
	}

	// ── Context length resolution ──────────────────────────────────
	resolver := NewModelContextResolver(in.ContextLookup)
	out.Context = resolver.Resolve(ModelContextQuery{
		Provider:  out.Provider,
		Model:     out.Model,
		BaseURL:   out.BaseURL,
		ModelInfo: in.ModelInfo,
	})

	// ── Evidence assembly ──────────────────────────────────────────
	out.Evidence = buildAzureFoundryEvidence(out, in.Probe)
	out.Degraded = out.BaseURL == "" || !out.KeyAvailable

	return out
}

func envValue(lookup func(string) (string, bool), name string) string {
	if lookup == nil {
		return ""
	}
	v, ok := lookup(name)
	if !ok {
		return ""
	}
	return v
}

func normalizeAzureBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func azureFoundryEffectiveModel(configModel, targetModel string) string {
	if target := strings.TrimSpace(targetModel); target != "" {
		return target
	}
	return strings.TrimSpace(configModel)
}

// redactAzureFoundryKey produces a short redacted preview of the API
// key suitable for status output. The full key never escapes the
// resolver. The preview shows the first 4 characters followed by a
// fixed redaction suffix so operators can visually confirm they are
// looking at the expected key without exposing usable material.
func redactAzureFoundryKey(key string) string {
	const previewLen = 4
	const ellipsis = "...redacted"
	if len(key) <= previewLen {
		return ellipsis
	}
	return key[:previewLen] + ellipsis
}

func buildAzureFoundryEvidence(rt AzureFoundryRuntime, probe AzureProbeResult) []string {
	evidence := make([]string, 0, 8)

	// Base URL evidence
	if rt.BaseURL == "" {
		evidence = append(evidence, "azure_foundry_base_url_missing")
	} else {
		evidence = append(evidence, fmt.Sprintf("azure_foundry_base_url_source=%s", rt.BaseURLSource))
	}

	// API key evidence - record either redaction (key present) or the
	// missing flag (key absent). Never both at once.
	if rt.KeyAvailable {
		evidence = append(evidence, fmt.Sprintf("azure_secret_redacted source=%s fingerprint=%s",
			rt.KeySource, rt.KeyFingerprint))
	} else {
		evidence = append(evidence, "azure_foundry_key_missing")
	}

	// API mode evidence (config-pinned, probe-supplied, inferred, or unknown)
	if rt.APIMode != "" && rt.APIMode != AzureTransportUnknown {
		evidence = append(evidence, fmt.Sprintf("azure_api_mode=%s source=%s", rt.APIMode, rt.APIModeSource))
	}
	switch {
	case rt.APIModeSource == AzureFoundryAPIModeSourceInferred:
		evidence = append(evidence, fmt.Sprintf("azure_foundry_api_mode_inferred model=%s mode=%s", rt.Model, rt.APIMode))
	case rt.APIMode != "" && rt.APIMode != AzureTransportUnknown:
		evidence = append(evidence, fmt.Sprintf("azure_foundry_api_mode_preserved mode=%s source=%s", rt.APIMode, rt.APIModeSource))
	default:
		evidence = append(evidence, "azure_foundry_api_mode_unknown")
	}

	// Probe evidence: forward transport and any advisory model IDs the
	// probe surfaced, but never persist them outside this read model.
	if probe.Transport != "" && probe.Transport != AzureTransportUnknown {
		evidence = append(evidence, fmt.Sprintf("azure_probe_transport=%s", probe.Transport))
	}
	for _, line := range probe.Evidence {
		if strings.HasPrefix(line, "model_id=") {
			evidence = append(evidence, line)
		}
	}

	// Context evidence
	if rt.Context.Known() {
		evidence = append(evidence, fmt.Sprintf("azure_context_known length=%d source=%s",
			rt.Context.ContextLength, rt.Context.Source))
	} else {
		evidence = append(evidence, "azure_context_unknown")
	}

	return evidence
}
