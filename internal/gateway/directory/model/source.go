package model

// Source is the minimal session-origin-shaped source needed to remember
// channel directory entries without depending on gateway runtime types.
type Source struct {
	Platform     string
	ChatID       string
	ChatName     string
	ChatType     string
	UserID       string
	UserName     string
	ThreadID     string
	ChatTopic    string
	GuildID      string
	ParentChatID string
	MessageID    string
}

// RememberedSourceEntry is the session-origin-shaped source record preserved
// for channel-directory refresh. Fields intentionally mirror Hermes session
// origin data plus enough metadata to produce Entry values.
type RememberedSourceEntry struct {
	Platform     string `json:"platform"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	ChatID       string `json:"chat_id,omitempty"`
	ChatName     string `json:"chat_name,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	UserName     string `json:"user_name,omitempty"`
	ThreadID     string `json:"thread_id,omitempty"`
	ChatTopic    string `json:"chat_topic,omitempty"`
	GuildID      string `json:"guild_id,omitempty"`
	ParentChatID string `json:"parent_chat_id,omitempty"`
	MessageID    string `json:"message_id,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// RememberedSourceLedger is the on-disk remembered-source ledger shape.
type RememberedSourceLedger struct {
	UpdatedAt string                             `json:"updated_at,omitempty"`
	Platforms map[string][]RememberedSourceEntry `json:"platforms"`
}

// EmptyRememberedSourceLedger returns a ledger with initialized platform buckets.
func EmptyRememberedSourceLedger() RememberedSourceLedger {
	return RememberedSourceLedger{Platforms: EmptyPlatformBuckets[RememberedSourceEntry]()}
}

// EnsureRememberedSourceLedger initializes the platform buckets after JSON decode.
func EnsureRememberedSourceLedger(ledger RememberedSourceLedger) RememberedSourceLedger {
	ledger.Platforms = EnsurePlatformBuckets(ledger.Platforms)
	return ledger
}

func RememberedSourceEntryFromSource(source Source) RememberedSourceEntry {
	entry := RememberedSourceEntry{
		Platform:     NormalizePlatform(source.Platform),
		Type:         normalizedSourceChatType(source),
		ChatID:       trimText(source.ChatID),
		ChatName:     trimText(source.ChatName),
		UserID:       trimText(source.UserID),
		UserName:     trimText(source.UserName),
		ThreadID:     trimText(source.ThreadID),
		GuildID:      trimText(source.GuildID),
		ParentChatID: trimText(source.ParentChatID),
		MessageID:    trimText(source.MessageID),
	}
	entry.ID = rememberedSourceID(entry)
	entry.Name = rememberedSourceName(entry)
	return NormalizeRememberedSourceEntry(entry)
}

func (e RememberedSourceEntry) ChannelDirectoryEntry() Entry {
	return Entry{
		ID:        e.ID,
		Name:      e.Name,
		Type:      e.Type,
		Guild:     e.GuildID,
		ChatID:    e.ChatID,
		ThreadID:  e.ThreadID,
		ChatTopic: e.ChatTopic,
	}
}

func NormalizeRememberedSourceEntry(entry RememberedSourceEntry) RememberedSourceEntry {
	entry.Platform = NormalizePlatform(entry.Platform)
	entry.ID = trimText(entry.ID)
	entry.Name = trimText(entry.Name)
	entry.Type = trimText(entry.Type)
	entry.ChatID = trimText(entry.ChatID)
	entry.ChatName = trimText(entry.ChatName)
	entry.UserID = trimText(entry.UserID)
	entry.UserName = trimText(entry.UserName)
	entry.ThreadID = trimText(entry.ThreadID)
	entry.ChatTopic = trimText(entry.ChatTopic)
	entry.GuildID = trimText(entry.GuildID)
	entry.ParentChatID = trimText(entry.ParentChatID)
	entry.MessageID = trimText(entry.MessageID)
	entry.UpdatedAt = trimText(entry.UpdatedAt)
	if entry.ID == "" {
		entry.ID = rememberedSourceID(entry)
	}
	if entry.Name == "" {
		entry.Name = rememberedSourceName(entry)
	}
	return entry
}

// UpsertRememberedSourceEntry replaces an existing remembered source with the
// same normalized ID or appends it when the session-discovered target is new.
// It returns false without changing entries when the source lacks the minimum
// remembered-source contract needed for refresh merges.
func UpsertRememberedSourceEntry(entries []RememberedSourceEntry, entry RememberedSourceEntry) ([]RememberedSourceEntry, bool) {
	return upsertByNormalizedID(entries, entry, NormalizeRememberedSourceEntry, func(entry RememberedSourceEntry) string {
		return entry.ID
	}, func(entry RememberedSourceEntry) bool {
		return entry.Platform != "" && entry.ID != ""
	})
}
