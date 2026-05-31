package model

import "strings"

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
	return strings.ToLower(trimText(platform))
}

// NormalizeEntry trims persisted target fields before merge and lookup.
func NormalizeEntry(entry Entry) Entry {
	entry.ID = trimText(entry.ID)
	entry.Name = trimText(entry.Name)
	entry.Type = trimText(entry.Type)
	entry.Guild = trimText(entry.Guild)
	entry.ChatID = trimText(entry.ChatID)
	entry.ThreadID = trimText(entry.ThreadID)
	entry.ChatTopic = trimText(entry.ChatTopic)
	return entry
}

// NormalizeQuery returns the canonical form used for human channel lookups.
func NormalizeQuery(value string) string {
	return strings.ToLower(trimLeadingChannelMarker(trimText(value)))
}

func trimLeadingChannelMarker(value string) string {
	return trimText(strings.TrimLeft(value, "#"))
}

// UpsertEntry replaces an existing entry with the same normalized ID or appends
// it when no cached target exists yet.
func UpsertEntry(entries []Entry, entry Entry) []Entry {
	entries, _ = UpsertValidEntry(entries, entry)
	return entries
}

// UpsertValidEntry normalizes and upserts a complete directory entry. It returns
// false without changing entries when the entry lacks the minimum cached-target
// contract used by refresh merges.
func UpsertValidEntry(entries []Entry, entry Entry) ([]Entry, bool) {
	return upsertByNormalizedID(entries, entry, NormalizeEntry, func(entry Entry) string {
		return entry.ID
	}, func(entry Entry) bool {
		return entry.ID != "" && entry.Name != ""
	})
}
