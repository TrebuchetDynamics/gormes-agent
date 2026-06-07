package startup

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// RedactRuntimeSecretText replaces configured startup secrets in operator-facing errors.
func RedactRuntimeSecretText(text string, secrets ...string) string {
	redacted := text
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")
	}
	return redacted
}

// FormatProviderSetupError renders the TUI startup provider setup guidance.
func FormatProviderSetupError(detail string, cfg config.Config, providerName, modelName string) string {
	providerName = strings.TrimSpace(providerName)
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = strings.TrimSpace(cfg.Hermes.Model)
	}
	return strings.Join([]string{
		"Gormes provider setup needed",
		"",
		"Startup cannot contact a model because provider settings are incomplete.",
		"",
		"Detected:",
		"  home:     " + config.GormesHome(),
		"  provider: " + SetupDisplayValue(providerName),
		"  model:    " + SetupDisplayValue(modelName),
		"",
		"Fix:",
		"  gormes setup model        choose provider/model defaults",
		"  gormes setup provider     add endpoint and API key",
		"  gormes auth add <provider>  add OAuth/API credentials when supported",
		"",
		"Smoke test without a provider:",
		"  gormes --offline",
		"",
		"Advanced config/env:",
		"  hermes.endpoint, hermes.provider, GORMES_ENDPOINT, GORMES_API_KEY",
		"",
		"Details:",
		"  " + FriendlyProviderSetupDetail(detail),
	}, "\n")
}

// SetupDisplayValue normalizes empty provider setup values for guidance output.
func SetupDisplayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "missing"
	}
	return value
}

// FriendlyProviderSetupDetail rewrites internal provider wording for setup guidance.
func FriendlyProviderSetupDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "endpoint unconfigured and no provider declared"):
		return "No provider endpoint or credential-backed provider is configured."
	case strings.Contains(lower, "endpoint unconfigured for provider"):
		return strings.ReplaceAll(detail, "hermes endpoint", "provider endpoint")
	default:
		return strings.ReplaceAll(detail, "hermes endpoint", "provider endpoint")
	}
}
