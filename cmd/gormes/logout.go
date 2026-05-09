package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const topLevelLogoutAllowedProviders = "nous|openai-codex|spotify"

func newLogoutCommand() *cobra.Command {
	var provider string
	cmd := &cobra.Command{
		Use:          "logout --provider <provider>",
		Short:        "Clear stored authentication for a Hermes-compatible provider",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTopLevelLogoutCommand(cmd, provider)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider to log out from: nous, openai-codex, or spotify")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, action, provider, redacted}")
	return cmd
}

func runTopLevelLogoutCommand(cmd *cobra.Command, providerInput string) error {
	provider := normalizeAuthProvider(providerInput)
	if !topLevelLogoutProviderAllowed(provider) {
		return newExitCodeError(2, fmt.Errorf("gormes logout: auth_logout_provider_unsupported provider=%s allowed=%s", provider, topLevelLogoutAllowedProviders))
	}
	return runAuthLogoutCommand(cmd, provider)
}

func topLevelLogoutProviderAllowed(provider string) bool {
	switch strings.TrimSpace(provider) {
	case "nous", "openai-codex", "spotify":
		return true
	default:
		return false
	}
}
