package gateway

import gatewayplatforms "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms"

func normalizedPlatformName(platform string) string {
	return gatewayplatforms.NormalizePlatformID(platform)
}

func platformBaseName(platform string) string {
	return gatewayplatforms.PlatformBaseName(platform)
}

func isPlatformName(platform, base string) bool {
	return gatewayplatforms.IsPlatformName(platform, base)
}

func isTelegramPlatform(platform string) bool {
	return gatewayplatforms.IsTelegramPlatform(platform)
}

func isDiscordPlatform(platform string) bool {
	return gatewayplatforms.IsDiscordPlatform(platform)
}

func isSlackPlatform(platform string) bool {
	return gatewayplatforms.IsSlackPlatform(platform)
}

func telegramDMTopicReplyFallbackLane(platform, chatID, threadID string) bool {
	return gatewayplatforms.TelegramDMTopicReplyFallbackLane(platform, chatID, threadID)
}

func defaultToolProgressModeForPlatform(platform string) string {
	return gatewayplatforms.DefaultToolProgressModeForPlatform(platform)
}
