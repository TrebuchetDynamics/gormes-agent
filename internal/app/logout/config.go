package logout

import "github.com/TrebuchetDynamics/gormes-agent/internal/config"

func ConfiguredProvider(normalize func(string) string) (string, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return "", err
	}
	provider := cfg.Hermes.Provider
	if normalize != nil {
		provider = normalize(provider)
	}
	return provider, nil
}

func ResetProviderIfMatching(provider string, normalize func(string) string) error {
	configured, err := ConfiguredProvider(normalize)
	if err != nil {
		return err
	}
	if configured != provider {
		return nil
	}
	return config.WriteTOMLValue(config.ConfigPath(), "hermes.provider", "auto")
}
