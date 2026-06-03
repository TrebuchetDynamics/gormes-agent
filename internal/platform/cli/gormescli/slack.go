package gormescli

import appslack "github.com/TrebuchetDynamics/gormes-agent/internal/app/slack"

func SlackManifestPayload(botName, description string, slashesOnly bool) (any, error) {
	return appslack.ManifestPayload(botName, description, slashesOnly)
}

func SlackManifestSlashCommands(requestURL string) []map[string]any {
	return appslack.ManifestSlashCommands(requestURL)
}

func SanitizeSlackManifestName(name string) string {
	return appslack.SanitizeManifestName(name)
}

func ClampString(s string, max int) string {
	return appslack.ClampString(s, max)
}

func NonEmpty(value, fallback string) string {
	return appslack.NonEmpty(value, fallback)
}
