// Package chatid parses transport evidence encoded in transcript chat_id values.
//
// It owns the pure chat_id -> source inference shared by session directory and
// ledger read models. It must never know about SQL stores, Bolt metadata, UI
// rendering, or command orchestration.
package chatid

import "strings"

// SourceFromTranscriptChatID returns the transport source encoded before the
// first colon in chatID. Empty or unqualified legacy chat IDs are CLI sessions.
func SourceFromTranscriptChatID(chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "cli"
	}
	if before, _, ok := strings.Cut(chatID, ":"); ok && strings.TrimSpace(before) != "" {
		return strings.ToLower(strings.TrimSpace(before))
	}
	return "cli"
}
