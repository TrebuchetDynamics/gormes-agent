package prompt

import "strings"

// IsSlashCommandText reports whether text starts with a slash command marker
// after operator-facing whitespace is ignored.
func IsSlashCommandText(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "/")
}

// SlashLeaksToModelPrompt reports whether the given inbound text would be
// forwarded to the model kernel as ordinary prompt content. Plain text leaks
// (that is the intended path); slash commands — recognized or not — are
// handled by the dispatcher and must not enter the prompt as command text.
func SlashLeaksToModelPrompt(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	return !IsSlashCommandText(text)
}
