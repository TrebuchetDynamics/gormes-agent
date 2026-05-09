package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const topLevelLogoutAllowedProviders = "nous|openai-codex|spotify"

func newLogoutCommand() *cobra.Command {
	var provider string
	cmd := &cobra.Command{
		Use:          "logout [--provider <provider>]",
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
	if provider == "" {
		var err error
		provider, err = topLevelLogoutDefaultProvider()
		if err != nil {
			return err
		}
		if provider == "" {
			return writeTopLevelLogoutDefaultAbsent(cmd)
		}
	}
	if !topLevelLogoutProviderAllowed(provider) {
		return newExitCodeError(2, fmt.Errorf("gormes logout: auth_logout_provider_unsupported provider=%s allowed=%s", provider, topLevelLogoutAllowedProviders))
	}
	if err := runAuthLogoutCommand(cmd, provider); err != nil {
		return err
	}
	return resetTopLevelLogoutProviderIfMatching(provider)
}

func topLevelLogoutProviderAllowed(provider string) bool {
	switch strings.TrimSpace(provider) {
	case "nous", "openai-codex", "spotify":
		return true
	default:
		return false
	}
}

func topLevelLogoutDefaultProvider() (string, error) {
	provider, err := topLevelLogoutConfiguredProvider()
	if err != nil {
		return "", err
	}
	switch provider {
	case "nous", "openai-codex":
		return provider, nil
	default:
		return "", nil
	}
}

func topLevelLogoutConfiguredProvider() (string, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return "", err
	}
	return normalizeAuthProvider(cfg.Hermes.Provider), nil
}

func resetTopLevelLogoutProviderIfMatching(provider string) error {
	configured, err := topLevelLogoutConfiguredProvider()
	if err != nil {
		return err
	}
	if configured != provider {
		return nil
	}
	return config.WriteTOMLValue(config.ConfigPath(), "hermes.provider", "auto")
}

func writeTopLevelLogoutDefaultAbsent(cmd *cobra.Command) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return writeAuthLifecycleJSON(cmd.OutOrStdout(), authLifecycleReportJSON{
			Build:    newBuildProvenance(),
			Action:   "absent",
			Provider: "auto",
			Redacted: true,
		})
	}
	fmt.Fprintln(cmd.OutOrStdout(), "auth_state_absent provider=auto redacted=true")
	return nil
}
