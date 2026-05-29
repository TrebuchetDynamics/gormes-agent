package platforms

import "strings"

func NormalizePlatformID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func PlatformBaseName(platform string) string {
	platform = NormalizePlatformID(platform)
	if before, _, ok := strings.Cut(platform, ":"); ok {
		return before
	}
	return platform
}

func IsPlatformName(platform, base string) bool {
	platform = NormalizePlatformID(platform)
	base = NormalizePlatformID(base)
	return platform == base || strings.HasPrefix(platform, base+":")
}

func IsTelegramPlatform(platform string) bool {
	return IsPlatformName(platform, "telegram")
}

func IsDiscordPlatform(platform string) bool {
	return IsPlatformName(platform, "discord")
}

func IsSlackPlatform(platform string) bool {
	return IsPlatformName(platform, "slack")
}
