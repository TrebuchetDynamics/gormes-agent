package providers

import applogout "github.com/TrebuchetDynamics/gormes-agent/internal/app/logout"

func TopLevelLogoutConfiguredProviderWithNormalizer(normalize func(string) string) (string, error) {
	return applogout.ConfiguredProvider(normalize)
}

func ResetTopLevelLogoutProviderIfMatchingWithNormalizer(provider string, normalize func(string) string) error {
	return applogout.ResetProviderIfMatching(provider, normalize)
}
