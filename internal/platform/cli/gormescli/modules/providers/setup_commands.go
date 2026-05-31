package providers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/provideridentity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// NewProvidersCommand creates an operator-facing provider setup guidance command.
func NewProvidersCommand(_ Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "providers [provider]",
		Aliases:      []string{"provider"},
		Short:        "Show provider setup commands",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && textvalue.IsNonBlank(args[0]) {
				return renderProviderSetupGuidance(cmd, args[0])
			}
			return renderAllProviderSetupGuidance(cmd)
		},
	}
	cmd.AddCommand(newProviderSetupGuidanceCommand())
	return cmd
}

func newProviderSetupGuidanceCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "setup [provider]",
		Short:        "Show provider setup commands",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && textvalue.IsNonBlank(args[0]) {
				return renderProviderSetupGuidance(cmd, args[0])
			}
			return renderAllProviderSetupGuidance(cmd)
		},
	}
}

func renderAllProviderSetupGuidance(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Provider setup commands:")
	fmt.Fprintln(out, "  Interactive: gormes setup provider")
	fmt.Fprintln(out, "  Model picker: gormes model")
	fmt.Fprintln(out, "  Credential pool: gormes auth add <provider>")
	fmt.Fprintln(out, "  Verify: gormes auth status <provider>")
	fmt.Fprintln(out, "Run `gormes providers setup <provider>` for provider-specific commands.")
	fmt.Fprintf(out, "Known manifest providers: %s\n", strings.Join(providerManifestIDs(), ", "))
	return nil
}

func renderProviderSetupGuidance(cmd *cobra.Command, rawProvider string) error {
	provider := textvalue.LowerTrim(rawProvider)
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	if !ok {
		return gormescli.NewExitCodeError(1, fmt.Errorf("unknown_provider: %s", provider))
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Provider setup commands for %s (%s):\n", providerDisplayName(entry.ID), entry.ID)
	fmt.Fprintf(out, "  Status: %s (%s, %s)\n", entry.ImplementationStatus, entry.TransportFamily, entry.AuthType)
	fmt.Fprintln(out, "  Interactive: gormes setup provider")
	if command := providerNonInteractiveSetupCommand(entry); command != "" {
		fmt.Fprintf(out, "  Non-interactive: %s\n", command)
	}
	for _, line := range providerCredentialGuidance(entry) {
		fmt.Fprintf(out, "  %s\n", line)
	}
	for _, line := range providerManualConfigGuidance(entry) {
		fmt.Fprintf(out, "  %s\n", line)
	}
	if model := providerDefaultModel(entry.ID); model != "" {
		fmt.Fprintf(out, "  Default model: %s\n", model)
	}
	fmt.Fprintln(out, "  Select model: gormes model")
	fmt.Fprintf(out, "  Verify auth: gormes auth status %s\n", entry.ID)
	fmt.Fprintln(out, "  Verify runtime: gormes doctor --offline")
	if len(entry.Aliases) > 0 {
		fmt.Fprintf(out, "  Aliases: %s\n", strings.Join(entry.Aliases, ", "))
	}
	if entry.ImplementationStatus == llm.ProviderRowBacked {
		fmt.Fprintln(out, "  Backlog: row-backed provider; setup commands record intent, but full runtime parity may still depend on the provider row.")
	}
	if entry.ImplementationStatus == llm.ProviderExcluded {
		fmt.Fprintln(out, "  Note: excluded provider; no runtime setup path is currently advertised.")
	}
	return nil
}

func providerManifestIDs() []string {
	entries := llm.HermesProviderRegistryManifest()
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	sort.Strings(ids)
	return ids
}

func providerDisplayName(provider string) string {
	return provideridentity.DisplayName(provider)
}

func providerNonInteractiveSetupCommand(entry llm.ProviderManifestEntry) string {
	if entry.ImplementationStatus == llm.ProviderExcluded || !providerSupportsNonInteractiveSetup(entry) {
		return ""
	}
	parts := []string{"GORMES_INFERENCE_PROVIDER=" + entry.ID}
	if providerNeedsExplicitEndpoint(entry) {
		parts = append(parts, "GORMES_ENDPOINT=<base-url>")
	}
	if providerNeedsAPIKeyForSetup(entry) {
		parts = append(parts, providerSetupAPIKeyEnv(entry)+"=<token>")
	}
	parts = append(parts, "gormes setup provider --non-interactive")
	return strings.Join(parts, " ")
}

func providerSupportsNonInteractiveSetup(entry llm.ProviderManifestEntry) bool {
	switch strings.TrimSpace(entry.AuthType) {
	case "api_key", "oauth_external", "oauth_device_code", "oauth_minimax":
		return true
	default:
		return false
	}
}

func providerNeedsExplicitEndpoint(entry llm.ProviderManifestEntry) bool {
	return !textvalue.IsNonBlank(entry.BaseURLOverride)
}

func providerNeedsAPIKeyForSetup(entry llm.ProviderManifestEntry) bool {
	switch strings.TrimSpace(entry.AuthType) {
	case "api_key":
		return true
	default:
		return false
	}
}

func providerSetupAPIKeyEnv(entry llm.ProviderManifestEntry) string {
	for _, envName := range entry.EnvVars {
		envName = strings.TrimSpace(envName)
		if envName != "" && !strings.EqualFold(envName, "CLAUDE_CODE_OAUTH_TOKEN") {
			return envName
		}
	}
	return "GORMES_API_KEY"
}

func providerCredentialGuidance(entry llm.ProviderManifestEntry) []string {
	switch strings.TrimSpace(entry.AuthType) {
	case "api_key":
		return providerAPIKeyCredentialGuidance(entry)
	case "aws_sdk":
		return []string{"Credentials: configure the AWS SDK credential chain; Bedrock does not use `gormes auth add`."}
	case "external_process":
		return []string{"Credentials: configure the external provider process, then select this provider with `gormes model` or config."}
	case "oauth_external", "oauth_device_code":
		if providerOAuthAdapterReady(entry.ID) {
			return []string{fmt.Sprintf("Credentials: gormes auth add %s --type oauth", entry.ID)}
		}
		return []string{"Credentials: OAuth adapter is row-backed; use `gormes config edit` only after the provider row lands."}
	case "oauth_minimax":
		return []string{"Credentials: MiniMax OAuth is row-backed; use `gormes providers setup minimax` or `minimax-cn` for API-key setup today."}
	default:
		return []string{fmt.Sprintf("Credentials: gormes auth add %s", entry.ID)}
	}
}

func providerManualConfigGuidance(entry llm.ProviderManifestEntry) []string {
	if entry.ImplementationStatus == llm.ProviderExcluded {
		return nil
	}
	lines := []string{fmt.Sprintf("Config: gormes config set hermes.provider %s", entry.ID)}
	endpoint := providerCredentialInferenceURL(entry)
	if endpoint == "" {
		endpoint = "<base-url>"
	}
	lines = append(lines, "Config: gormes config set hermes.endpoint "+endpoint)
	model := providerDefaultModel(entry.ID)
	if model == "" {
		model = "<model>"
	}
	lines = append(lines, "Config: gormes config set hermes.model "+model)
	return lines
}

func providerAPIKeyCredentialGuidance(entry llm.ProviderManifestEntry) []string {
	cmd := fmt.Sprintf("Credentials: gormes auth add %s --type api-key --api-key <token>", entry.ID)
	if endpoint := providerCredentialInferenceURL(entry); endpoint != "" {
		cmd += " --inference-url " + endpoint
	} else {
		cmd += " --inference-url <base-url>"
	}
	lines := []string{cmd}
	if providerOAuthAdapterReady(entry.ID) {
		lines = append(lines, fmt.Sprintf("OAuth alternative: gormes auth add %s --type oauth", entry.ID))
	}
	return lines
}

func providerCredentialInferenceURL(entry llm.ProviderManifestEntry) string {
	if endpoint := strings.TrimRight(strings.TrimSpace(entry.BaseURLOverride), "/"); endpoint != "" {
		return endpoint
	}
	return ""
}

func providerOAuthAdapterReady(provider string) bool {
	switch textvalue.LowerTrim(provider) {
	case "anthropic", "nous", "openai-codex", "google-gemini-cli", "qwen-oauth":
		return true
	default:
		return false
	}
}

func providerDefaultModel(provider string) string {
	resolution := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{})
	return strings.TrimSpace(resolution.Model)
}
