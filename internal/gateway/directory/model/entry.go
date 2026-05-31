package model

import (
	"strings"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
)

// Entry describes one platform target that can be selected by exact ID, human
// display name, guild-qualified name, or type/display lookup.
type Entry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	Guild     string `json:"guild,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"`
	ChatTopic string `json:"chat_topic,omitempty"`
}

// Evidence carries user-safe degraded-mode evidence without leaking local state paths.
type Evidence struct {
	Code     string
	Platform string
	Query    string
}

// NormalizePlatform returns the canonical platform key used by directory stores,
// refresh merges, and channel lookups.
func NormalizePlatform(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

// NormalizeEntry trims persisted target fields before merge and lookup.
func NormalizeEntry(entry Entry) Entry {
	entry.ID = strings.TrimSpace(entry.ID)
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Type = strings.TrimSpace(entry.Type)
	entry.Guild = strings.TrimSpace(entry.Guild)
	entry.ChatID = strings.TrimSpace(entry.ChatID)
	entry.ThreadID = strings.TrimSpace(entry.ThreadID)
	entry.ChatTopic = strings.TrimSpace(entry.ChatTopic)
	return entry
}

// NormalizeQuery returns the canonical form used for human channel lookups.
func NormalizeQuery(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimLeft(value, "#")))
}

// UpsertEntry replaces an existing entry with the same normalized ID or appends
// it when no cached target exists yet.
func UpsertEntry(entries []Entry, entry Entry) []Entry {
	entry = NormalizeEntry(entry)
	for i, existing := range entries {
		if strings.TrimSpace(existing.ID) == entry.ID {
			entries[i] = entry
			return entries
		}
	}
	return append(entries, entry)
}

// DeliveryTarget converts a directory entry into the gateway delivery target contract.
func DeliveryTarget(platform string, entry Entry) gatewaydelivery.Target {
	chatID := strings.TrimSpace(entry.ChatID)
	threadID := strings.TrimSpace(entry.ThreadID)
	if chatID == "" {
		parts := strings.SplitN(strings.TrimSpace(entry.ID), ":", 2)
		chatID = parts[0]
		if len(parts) == 2 && threadID == "" {
			threadID = parts[1]
		}
	}
	return gatewaydelivery.Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}
}
