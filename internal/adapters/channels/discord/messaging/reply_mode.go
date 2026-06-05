package messaging

import "strings"

func NormalizeReplyToMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off":
		return "off"
	case "all":
		return "all"
	default:
		return "first"
	}
}

func IsMissingDiscordReplyReference(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "error code: 10008") ||
		(strings.Contains(text, "error code: 50035") && strings.Contains(text, "Cannot reply to a system message"))
}
