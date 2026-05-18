package main

import (
	"github.com/spf13/cobra"

	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const topLevelLogoutAllowedProviders = providermodule.TopLevelLogoutAllowedProviders

type logoutCommandSeams = providermodule.LogoutSeams

func newLogoutCommand() *cobra.Command {
	return providermodule.NewLogoutCommandWithSeams(defaultLogoutCommandSeams(), providerCommandOptions())
}

func runTopLevelLogoutCommand(cmd *cobra.Command, providerInput string) error {
	return providermodule.RunTopLevelLogoutCommand(cmd, providerInput, defaultLogoutCommandSeams(), providerCommandOptions())
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
	return providermodule.TopLevelLogoutProviderAllowed(provider)
}

func topLevelLogoutDefaultProvider() (string, error) {
	return providermodule.TopLevelLogoutDefaultProvider(defaultLogoutCommandSeams())
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
	return providermodule.RunTopLevelLogoutCommand(cmd, "", logoutCommandSeams{
		NormalizeAuthProvider: func(string) string { return "" },
		ConfiguredProvider:    func() (string, error) { return "", nil },
		RunAuthLogout:         func(*cobra.Command, string) error { return nil },
	}, providerCommandOptions())
}
