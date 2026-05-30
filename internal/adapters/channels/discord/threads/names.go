package threads

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const DiscordThreadNameLimit = 80

var discordMentionTokenRE = regexp.MustCompile(`<@[!&]?\d+>|<#\d+>`)

// SanitizeDiscordThreadName returns a Discord-safe thread name with mentions removed.
func SanitizeDiscordThreadName(name string) string {
	name = discordMentionTokenRE.ReplaceAllString(strings.TrimSpace(name), "")
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = "Gormes"
	}
	if utf8.RuneCountInString(name) <= DiscordThreadNameLimit {
		return name
	}
	runes := []rune(name)
	if DiscordThreadNameLimit <= 3 {
		return string(runes[:DiscordThreadNameLimit])
	}
	return strings.TrimSpace(string(runes[:DiscordThreadNameLimit-3])) + "..."
}

// NormalizeDiscordArchiveDuration returns a Discord-supported auto-archive duration.
func NormalizeDiscordArchiveDuration(value int) int {
	switch value {
	case 60, 1440, 4320, 10080:
		return value
	default:
		return 1440
	}
}
