package setupguidance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/provideridentity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// ManifestIDs returns the sorted provider IDs advertised by the Hermes provider manifest.
func ManifestIDs() []string {
	entries := llm.HermesProviderRegistryManifest()
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	sort.Strings(ids)
	return ids
}

// DisplayName returns the operator-facing display name for a provider ID.
func DisplayName(provider string) string {
	return provideridentity.DisplayName(provider)
}

// NonInteractiveSetupCommand returns the non-interactive setup command for a manifest entry.
func NonInteractiveSetupCommand(entry llm.ProviderManifestEntry) string {
	if entry.ImplementationStatus == llm.ProviderExcluded || !supportsNonInteractiveSetup(entry) {
		return ""
	}
	parts := []string{"GORMES_INFERENCE_PROVIDER=" + entry.ID}
	if needsExplicitEndpoint(entry) {
		parts = append(parts, "GORMES_ENDPOINT=<base-url>")
	}
	if needsAPIKeyForSetup(entry) {
		parts = append(parts, setupAPIKeyEnv(entry)+"=<token>")
	}
	parts = append(parts, "gormes setup provider --non-interactive")
	return strings.Join(parts, " ")
}

func supportsNonInteractiveSetup(entry llm.ProviderManifestEntry) bool {
	switch strings.TrimSpace(entry.AuthType) {
	case "api_key", "oauth_external", "oauth_device_code", "oauth_minimax":
		return true
	default:
		return false
	}
}

func needsExplicitEndpoint(entry llm.ProviderManifestEntry) bool {
	return !textvalue.IsNonBlank(entry.BaseURLOverride)
}

func needsAPIKeyForSetup(entry llm.ProviderManifestEntry) bool {
	switch strings.TrimSpace(entry.AuthType) {
	case "api_key":
		return true
	default:
		return false
	}
}

func setupAPIKeyEnv(entry llm.ProviderManifestEntry) string {
	for _, envName := range entry.EnvVars {
		envName = strings.TrimSpace(envName)
		if envName != "" && !strings.EqualFold(envName, "CLAUDE_CODE_OAUTH_TOKEN") {
			return envName
		}
	}
	return "GORMES_API_KEY"
}

// CredentialGuidance returns provider-specific credential setup guidance lines.
func CredentialGuidance(entry llm.ProviderManifestEntry) []string {
	switch strings.TrimSpace(entry.AuthType) {
	case "api_key":
		return apiKeyCredentialGuidance(entry)
	case "aws_sdk":
		return []string{"Credentials: configure the AWS SDK credential chain; Bedrock does not use `gormes auth add`."}
	case "external_process":
		return []string{"Credentials: configure the external provider process, then select this provider with `gormes model` or config."}
	case "oauth_external", "oauth_device_code":
		if oauthAdapterReady(entry.ID) {
			return []string{fmt.Sprintf("Credentials: gormes auth add %s --type oauth", entry.ID)}
		}
		return []string{"Credentials: OAuth adapter is row-backed; use `gormes config edit` only after the provider row lands."}
	case "oauth_minimax":
		return []string{"Credentials: MiniMax OAuth is row-backed; use `gormes providers setup minimax` or `minimax-cn` for API-key setup today."}
	default:
		return []string{fmt.Sprintf("Credentials: gormes auth add %s", entry.ID)}
	}
}

// ManualConfigGuidance returns manual config commands for a manifest entry.
func ManualConfigGuidance(entry llm.ProviderManifestEntry) []string {
	if entry.ImplementationStatus == llm.ProviderExcluded {
		return nil
	}
	lines := []string{fmt.Sprintf("Config: gormes config set hermes.provider %s", entry.ID)}
	endpoint := credentialInferenceURL(entry)
	if endpoint == "" {
		endpoint = "<base-url>"
	}
	lines = append(lines, "Config: gormes config set hermes.endpoint "+endpoint)
	model := DefaultModel(entry.ID)
	if model == "" {
		model = "<model>"
	}
	lines = append(lines, "Config: gormes config set hermes.model "+model)
	return lines
}

func apiKeyCredentialGuidance(entry llm.ProviderManifestEntry) []string {
	cmd := fmt.Sprintf("Credentials: gormes auth add %s --type api-key --api-key <token>", entry.ID)
	if endpoint := credentialInferenceURL(entry); endpoint != "" {
		cmd += " --inference-url " + endpoint
	} else {
		cmd += " --inference-url <base-url>"
	}
	lines := []string{cmd}
	if oauthAdapterReady(entry.ID) {
		lines = append(lines, fmt.Sprintf("OAuth alternative: gormes auth add %s --type oauth", entry.ID))
	}
	return lines
}

func credentialInferenceURL(entry llm.ProviderManifestEntry) string {
	if endpoint := strings.TrimRight(strings.TrimSpace(entry.BaseURLOverride), "/"); endpoint != "" {
		return endpoint
	}
	return ""
}

func oauthAdapterReady(provider string) bool {
	switch textvalue.LowerTrim(provider) {
	case "anthropic", "nous", "openai-codex", "google-gemini-cli", "qwen-oauth":
		return true
	default:
		return false
	}
}

// DefaultModel returns the default model for a provider from the provider registry.
func DefaultModel(provider string) string {
	resolution := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{})
	return strings.TrimSpace(resolution.Model)
}
