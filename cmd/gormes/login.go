package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const topLevelLoginAllowedProviders = "nous|openai-codex"

func newLoginCommand() *cobra.Command {
	var provider string
	var portalURL string
	var inferenceURL string
	var clientID string
	var scope string
	var noBrowser bool
	var timeout string
	var insecure bool
	var caBundle string

	cmd := &cobra.Command{
		Use:          "login --provider <provider>",
		Short:        "Authenticate with a Hermes-compatible OAuth provider",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTopLevelLoginCommand(cmd, authAddOptions{
				Provider:     provider,
				AuthType:     config.CredentialAuthOAuth,
				PortalURL:    portalURL,
				InferenceURL: inferenceURL,
				ClientID:     clientID,
				Scope:        scope,
				NoBrowser:    noBrowser,
				Timeout:      timeout,
				Insecure:     insecure,
				CABundle:     caBundle,
			})
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider to log in to: nous or openai-codex")
	cmd.Flags().StringVar(&portalURL, "portal-url", "", "OAuth portal base URL")
	cmd.Flags().StringVar(&inferenceURL, "inference-url", "", "provider inference base URL override")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&scope, "scope", "", "OAuth scope")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open a browser for OAuth")
	cmd.Flags().StringVar(&timeout, "timeout", "", "OAuth timeout")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "disable OAuth TLS verification")
	cmd.Flags().StringVar(&caBundle, "ca-bundle", "", "OAuth CA bundle")
	return cmd
}

func runTopLevelLoginCommand(cmd *cobra.Command, opts authAddOptions) error {
	provider := normalizeAuthProvider(opts.Provider)
	if provider == "" {
		return newExitCodeError(2, fmt.Errorf("gormes login: auth_login_provider_required allowed=%s; use `gormes login --provider openai-codex` or `gormes auth add openai-codex --type oauth`", topLevelLoginAllowedProviders))
	}
	if !topLevelLoginProviderAllowed(provider) {
		return newExitCodeError(2, fmt.Errorf("gormes login: auth_login_provider_unsupported allowed=%s; use `gormes auth add <provider> --type oauth`", topLevelLoginAllowedProviders))
	}
	opts.Provider = provider
	opts.AuthType = config.CredentialAuthOAuth
	return runAuthAddCommand(cmd, opts)
}

func topLevelLoginProviderAllowed(provider string) bool {
	switch strings.TrimSpace(provider) {
	case "nous", config.CodexOAuthProvider:
		return true
	default:
		return false
	}
}
