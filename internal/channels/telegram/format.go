package telegram

import "regexp"

var (
	telegramMarkdownV2EscapeRE        = regexp.MustCompile("\\\\([_*\\[\\]()~`>#+\\-=|{}.!\\\\])")
	telegramMarkdownV2BoldRE          = regexp.MustCompile("\\*([^*\\n]+)\\*")
	telegramMarkdownV2ItalicRE        = regexp.MustCompile("(^|[^A-Za-z0-9_])_([^_\\n]+)_([^A-Za-z0-9_]|$)")
	telegramMarkdownV2StrikethroughRE = regexp.MustCompile("~([^~\\n]+)~")
	telegramMarkdownV2SpoilerRE       = regexp.MustCompile("\\|\\|([^|\\n]+)\\|\\|")
)

func stripTelegramMarkdownV2(text string) string {
	cleaned := telegramMarkdownV2EscapeRE.ReplaceAllString(text, "$1")
	cleaned = telegramMarkdownV2BoldRE.ReplaceAllString(cleaned, "$1")
	cleaned = telegramMarkdownV2ItalicRE.ReplaceAllString(cleaned, "${1}${2}${3}")
	cleaned = telegramMarkdownV2StrikethroughRE.ReplaceAllString(cleaned, "$1")
	cleaned = telegramMarkdownV2SpoilerRE.ReplaceAllString(cleaned, "$1")
	return cleaned
}
