package platforms

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/identity"

func NormalizePlatformID(value string) string {
	return identity.NormalizePlatformID(value)
}

func PlatformBaseName(platform string) string {
	return identity.PlatformBaseName(platform)
}

func IsPlatformName(platform, base string) bool {
	return identity.IsPlatformName(platform, base)
}

func IsTelegramPlatform(platform string) bool {
	return identity.IsTelegramPlatform(platform)
}

func IsDiscordPlatform(platform string) bool {
	return identity.IsDiscordPlatform(platform)
}

func IsSlackPlatform(platform string) bool {
	return identity.IsSlackPlatform(platform)
}

func TelegramDMTopicReplyFallbackLane(platform, chatID, threadID string) bool {
	return identity.TelegramDMTopicReplyFallbackLane(platform, chatID, threadID)
}

func DefaultToolProgressModeForPlatform(platform string) string {
	return identity.DefaultToolProgressModeForPlatform(platform)
}
