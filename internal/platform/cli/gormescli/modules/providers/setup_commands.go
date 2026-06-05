package providers

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/setupguidance"
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
	fmt.Fprintf(out, "Known manifest providers: %s\n", strings.Join(setupguidance.ManifestIDs(), ", "))
	return nil
}

func renderProviderSetupGuidance(cmd *cobra.Command, rawProvider string) error {
	provider := textvalue.LowerTrim(rawProvider)
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	if !ok {
		return gormescli.NewExitCodeError(1, fmt.Errorf("unknown_provider: %s", provider))
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Provider setup commands for %s (%s):\n", setupguidance.DisplayName(entry.ID), entry.ID)
	fmt.Fprintf(out, "  Status: %s (%s, %s)\n", entry.ImplementationStatus, entry.TransportFamily, entry.AuthType)
	fmt.Fprintln(out, "  Interactive: gormes setup provider")
	if command := setupguidance.NonInteractiveSetupCommand(entry); command != "" {
		fmt.Fprintf(out, "  Non-interactive: %s\n", command)
	}
	for _, line := range setupguidance.CredentialGuidance(entry) {
		fmt.Fprintf(out, "  %s\n", line)
	}
	for _, line := range setupguidance.ManualConfigGuidance(entry) {
		fmt.Fprintf(out, "  %s\n", line)
	}
	if model := setupguidance.DefaultModel(entry.ID); model != "" {
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
