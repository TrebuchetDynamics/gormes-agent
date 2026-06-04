package main

import (
	"github.com/spf13/cobra"

	logoutcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/logout"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const topLevelLogoutAllowedProviders = logoutcmd.TopLevelLogoutAllowedProviders

type logoutCommandSeams = logoutcmd.LogoutSeams

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
	return logoutcmd.RunTopLevelLogoutCommand(cmd, providerInput, defaultLogoutCommandSeams(), logoutCommandOptions())
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
	return logoutcmd.TopLevelLogoutProviderAllowed(provider)
}

func topLevelLogoutDefaultProvider() (string, error) {
	return logoutcmd.TopLevelLogoutDefaultProvider(defaultLogoutCommandSeams())
}

func topLevelLogoutConfiguredProvider() (string, error) {
	return logoutcmd.ConfiguredProvider(normalizeAuthProvider)
}

func resetTopLevelLogoutProviderIfMatching(provider string) error {
	return logoutcmd.ResetProviderIfMatching(provider, normalizeAuthProvider)
}

func writeTopLevelLogoutDefaultAbsent(cmd *cobra.Command) error {
	return logoutcmd.RunTopLevelLogoutCommand(cmd, "", logoutCommandSeams{
		NormalizeAuthProvider: func(string) string { return "" },
		ConfiguredProvider:    func() (string, error) { return "", nil },
		RunAuthLogout:         func(*cobra.Command, string) error { return nil },
	}, logoutCommandOptions())
}

func logoutCommandOptions() logoutcmd.Options {
	return logoutcmd.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}
