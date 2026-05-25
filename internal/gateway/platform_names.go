package gateway

import "strings"

func normalizedPlatformName(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func platformBaseName(platform string) string {
	platform = normalizedPlatformName(platform)
	if before, _, ok := strings.Cut(platform, ":"); ok {
		return before
	}
	return platform
}

func isPlatformName(platform, base string) bool {
	platform = normalizedPlatformName(platform)
	base = normalizedPlatformName(base)
	return platform == base || strings.HasPrefix(platform, base+":")
}

func isTelegramPlatform(platform string) bool {
	return isPlatformName(platform, "telegram")
}

func isDiscordPlatform(platform string) bool {
	return isPlatformName(platform, "discord")
}

func isSlackPlatform(platform string) bool {
	return isPlatformName(platform, "slack")
}
