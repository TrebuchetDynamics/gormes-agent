package model

import "strings"

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
	return RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}
}

// EnsureRememberedSourceLedger initializes the platform buckets after JSON decode.
func EnsureRememberedSourceLedger(ledger RememberedSourceLedger) RememberedSourceLedger {
	if ledger.Platforms == nil {
		ledger.Platforms = map[string][]RememberedSourceEntry{}
	}
	return ledger
}

func RememberedSourceEntryFromSource(source Source) RememberedSourceEntry {
	entry := RememberedSourceEntry{
		Platform:     NormalizePlatform(source.Platform),
		Type:         normalizedSourceChatType(source),
		ChatID:       strings.TrimSpace(source.ChatID),
		ChatName:     strings.TrimSpace(source.ChatName),
		UserID:       strings.TrimSpace(source.UserID),
		UserName:     strings.TrimSpace(source.UserName),
		ThreadID:     strings.TrimSpace(source.ThreadID),
		GuildID:      strings.TrimSpace(source.GuildID),
		ParentChatID: strings.TrimSpace(source.ParentChatID),
		MessageID:    strings.TrimSpace(source.MessageID),
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
	entry.ID = strings.TrimSpace(entry.ID)
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Type = strings.TrimSpace(entry.Type)
	entry.ChatID = strings.TrimSpace(entry.ChatID)
	entry.ChatName = strings.TrimSpace(entry.ChatName)
	entry.UserID = strings.TrimSpace(entry.UserID)
	entry.UserName = strings.TrimSpace(entry.UserName)
	entry.ThreadID = strings.TrimSpace(entry.ThreadID)
	entry.ChatTopic = strings.TrimSpace(entry.ChatTopic)
	entry.GuildID = strings.TrimSpace(entry.GuildID)
	entry.ParentChatID = strings.TrimSpace(entry.ParentChatID)
	entry.MessageID = strings.TrimSpace(entry.MessageID)
	entry.UpdatedAt = strings.TrimSpace(entry.UpdatedAt)
	if entry.ID == "" {
		entry.ID = rememberedSourceID(entry)
	}
	if entry.Name == "" {
		entry.Name = rememberedSourceName(entry)
	}
	return entry
}

func rememberedSourceID(entry RememberedSourceEntry) string {
	chatID := strings.TrimSpace(entry.ChatID)
	if chatID == "" {
		return ""
	}
	if threadID := strings.TrimSpace(entry.ThreadID); threadID != "" {
		return chatID + ":" + threadID
	}
	return chatID
}

func rememberedSourceName(entry RememberedSourceEntry) string {
	base := strings.TrimSpace(entry.ChatName)
	if base == "" {
		base = strings.TrimSpace(entry.UserName)
	}
	if base == "" {
		base = strings.TrimSpace(entry.ChatID)
	}
	if threadID := strings.TrimSpace(entry.ThreadID); threadID != "" {
		topic := strings.TrimSpace(entry.ChatTopic)
		if topic == "" {
			topic = "topic " + threadID
		}
		return base + " / " + topic
	}
	return base
}

func normalizedSourceChatType(source Source) string {
	if chatType := strings.TrimSpace(source.ChatType); chatType != "" {
		return chatType
	}
	if strings.TrimSpace(source.ThreadID) != "" {
		return "thread"
	}
	return "dm"
}
