package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const topLevelLogoutAllowedProviders = gormescli.TopLevelLogoutAllowedProviders

type logoutCommandSeams = gormescli.LogoutSeams

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
	return gormescli.RunTopLevelLogoutCommand(cmd, providerInput, defaultLogoutCommandSeams(), logoutCommandOptions())
}

func defaultLogoutCommandSeams() logoutCommandSeams {
	return logoutCommandSeams{
		NormalizeAuthProvider:   normalizeAuthProvider,
		ConfiguredProvider:      topLevelLogoutConfiguredProvider,
		RunAuthLogout:           runAuthLogoutCommand,
		ResetProviderIfMatching: resetTopLevelLogoutProviderIfMatching,
	}
}

func topLevelLogoutProviderAllowed(provider string) bool {
	return gormescli.TopLevelLogoutProviderAllowed(provider)
}

func topLevelLogoutDefaultProvider() (string, error) {
	return gormescli.TopLevelLogoutDefaultProvider(defaultLogoutCommandSeams())
}

func topLevelLogoutConfiguredProvider() (string, error) {
	return gormescli.ConfiguredLogoutProvider(normalizeAuthProvider)
}

func resetTopLevelLogoutProviderIfMatching(provider string) error {
	return gormescli.ResetLogoutProviderIfMatching(provider, normalizeAuthProvider)
}

func writeTopLevelLogoutDefaultAbsent(cmd *cobra.Command) error {
	return gormescli.RunTopLevelLogoutCommand(cmd, "", logoutCommandSeams{
		NormalizeAuthProvider: func(string) string { return "" },
		ConfiguredProvider:    func() (string, error) { return "", nil },
		RunAuthLogout:         func(*cobra.Command, string) error { return nil },
	}, logoutCommandOptions())
}

func logoutCommandOptions() gormescli.LogoutOptions {
	return gormescli.LogoutOptions{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}
