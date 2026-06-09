package llm

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// AgentIdentityForProfile returns the system identity for an active profile.
// Default/root profiles keep the stock Gorm persona; named profiles become the
// assistant's operator-visible name so the TUI prompt and model identity agree.
func AgentIdentityForProfile(profileName string) string {
	name := profileAssistantName(profileName)
	if name == "" {
		return DefaultAgentIdentity
	}
	return "You are " + name + ", an AI assistant run by gormes, a Go-native Hermes-compatible agent runtime. If asked your name, answer " + name + ". You are helpful, knowledgeable, and direct. You assist users with a wide range of tasks including answering questions, writing and editing code, analyzing information, creative work, and executing actions via your tools. You communicate clearly, admit uncertainty when appropriate, and prioritize being genuinely useful over being verbose unless otherwise directed below. Be targeted and efficient in your exploration and investigations."
}

func profileAssistantName(profileName string) string {
	profile := strings.TrimSpace(profileName)
	if profile == "" {
		return ""
	}
	switch strings.ToLower(profile) {
	case "default", "main", "root":
		return ""
	}
	return titleFirstRune(profile)
}

func titleFirstRune(value string) string {
	r, size := utf8.DecodeRuneInString(value)
	if r == utf8.RuneError && size == 0 {
		return ""
	}
	return string(unicode.ToTitle(r)) + value[size:]
}
