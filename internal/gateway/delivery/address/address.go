package address

import (
	"strconv"
	"strings"
)

// Platform normalizes channel names for delivery routing comparisons and map
// keys while keeping empty values empty for caller-specific validation.
func Platform(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// ID trims channel/chat/thread/user identifiers without changing their case or
// punctuation, preserving provider-owned identifier semantics.
func ID(value string) string {
	return strings.TrimSpace(value)
}

// ChatWithThread returns the session metadata chat key used for threaded
// delivery mirrors.
func ChatWithThread(chatID, threadID string) string {
	chatID = ID(chatID)
	threadID = ID(threadID)
	if threadID == "" {
		if strings.Contains(chatID, ":") {
			return strconv.Itoa(len(chatID)) + ":" + chatID + ":0:"
		}
		return chatID
	}
	if strings.Contains(chatID, ":") || strings.Contains(threadID, ":") {
		return strconv.Itoa(len(chatID)) + ":" + chatID + ":" + strconv.Itoa(len(threadID)) + ":" + threadID
	}
	return chatID + ":" + threadID
}

// ChatMatches reports whether a stored session chat key matches the requested
// chat/thread delivery address.
func ChatMatches(candidate, chatID, threadID string) bool {
	candidate = ID(candidate)
	chatID = ID(chatID)
	threadID = ID(threadID)
	if threadID == "" && candidate == chatID {
		return true
	}
	return candidate == ChatWithThread(chatID, threadID)
}
