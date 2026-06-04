package gormescli

import (
	"github.com/spf13/cobra"

	applogout "github.com/TrebuchetDynamics/gormes-agent/internal/app/logout"
)

const TopLevelLogoutAllowedProviders = applogout.TopLevelLogoutAllowedProviders

type LogoutOptions = applogout.Options

type LogoutSeams = applogout.LogoutSeams

type AuthLifecycleReportJSON = applogout.AuthLifecycleReportJSON

type AuthLifecycleRemovedJSON = applogout.RemovedJSON

func WriteAuthLifecycleJSON(out interface{ Write(p []byte) (int, error) }, report AuthLifecycleReportJSON) error {
	return applogout.WriteAuthLifecycleJSON(out, report)
}

func RunTopLevelLogoutCommand(cmd *cobra.Command, providerInput string, seams LogoutSeams, opts LogoutOptions) error {
	return applogout.RunTopLevelLogoutCommand(cmd, providerInput, seams, opts)
}

func TopLevelLogoutProviderAllowed(provider string) bool {
	return applogout.TopLevelLogoutProviderAllowed(provider)
}

func TopLevelLogoutDefaultProvider(seams LogoutSeams) (string, error) {
	return applogout.TopLevelLogoutDefaultProvider(seams)
}

func ConfiguredLogoutProvider(normalize func(string) string) (string, error) {
	return applogout.ConfiguredProvider(normalize)
}

func ResetLogoutProviderIfMatching(provider string, normalize func(string) string) error {
	return applogout.ResetProviderIfMatching(provider, normalize)
}
