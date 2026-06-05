package identity

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

func TelegramDMTopicReplyFallbackLane(platform, chatID, threadID string) bool {
	if !IsTelegramPlatform(platform) {
		return false
	}
	if strings.TrimSpace(threadID) == "" {
		return false
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return false
	}
	return !strings.HasPrefix(chatID, "-")
}

func DefaultToolProgressModeForPlatform(platform string) string {
	if IsTelegramPlatform(platform) || IsDiscordPlatform(platform) {
		return "all"
	}
	if IsSlackPlatform(platform) {
		return "off"
	}
	switch PlatformBaseName(platform) {
	case "api_server":
		return "all"
	case "mattermost", "matrix", "feishu", "whatsapp":
		return "new"
	case "signal", "bluebubbles", "weixin", "wecom", "wecom_callback", "dingtalk",
		"email", "sms", "webhook", "homeassistant":
		return "off"
	default:
		return "all"
	}
}
