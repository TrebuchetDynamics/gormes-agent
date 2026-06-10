package source

import (
	"strings"

	entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"
)

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
	return RememberedSourceLedger{Platforms: policy.EmptyPlatformBuckets[RememberedSourceEntry]()}
}

// EnsureRememberedSourceLedger initializes and normalizes platform buckets after
// JSON decode so older/operator-edited ledgers keep the same lookup contract as
// entries written through RememberSource.
func EnsureRememberedSourceLedger(ledger RememberedSourceLedger) RememberedSourceLedger {
	ledger.UpdatedAt = strings.Join(strings.Fields(ledger.UpdatedAt), " ")
	platforms := policy.EmptyPlatformBuckets[RememberedSourceEntry]()
	for platform, entries := range policy.EnsurePlatformBuckets(ledger.Platforms) {
		platform = entrymodel.NormalizePlatform(platform)
		if platform == "" {
			continue
		}
		for _, entry := range entries {
			if policy.TrimText(entry.Platform) == "" {
				entry.Platform = platform
			}
			entry = NormalizeRememberedSourceEntry(entry)
			if entry.Platform == "" || entry.ID == "" {
				continue
			}
			platforms[entry.Platform], _ = UpsertRememberedSourceEntry(platforms[entry.Platform], entry)
		}
	}
	ledger.Platforms = platforms
	return ledger
}

func RememberedSourceEntryFromSource(source Source) RememberedSourceEntry {
	entry := RememberedSourceEntry{
		Platform:     entrymodel.NormalizePlatform(source.Platform),
		Type:         normalizedSourceChatType(source),
		ChatID:       policy.TrimText(source.ChatID),
		ChatName:     policy.TrimText(source.ChatName),
		UserID:       policy.TrimText(source.UserID),
		UserName:     policy.TrimText(source.UserName),
		ThreadID:     policy.TrimText(source.ThreadID),
		ChatTopic:    policy.TrimText(source.ChatTopic),
		GuildID:      policy.TrimText(source.GuildID),
		ParentChatID: policy.TrimText(source.ParentChatID),
		MessageID:    policy.TrimText(source.MessageID),
	}
	entry.ID = rememberedSourceID(entry)
	entry.Name = rememberedSourceName(entry)
	return NormalizeRememberedSourceEntry(entry)
}

func (e RememberedSourceEntry) ChannelDirectoryEntry() entrymodel.Entry {
	return entrymodel.Entry{
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
	entry.Platform = entrymodel.NormalizePlatform(entry.Platform)
	entry.ID = policy.TrimText(entry.ID)
	entry.Name = policy.TrimText(entry.Name)
	entry.Type = policy.TrimText(entry.Type)
	entry.ChatID = policy.TrimText(entry.ChatID)
	entry.ChatName = policy.TrimText(entry.ChatName)
	entry.UserID = policy.TrimText(entry.UserID)
	entry.UserName = policy.TrimText(entry.UserName)
	entry.ThreadID = policy.TrimText(entry.ThreadID)
	entry.ChatTopic = policy.TrimText(entry.ChatTopic)
	entry.GuildID = policy.TrimText(entry.GuildID)
	entry.ParentChatID = policy.TrimText(entry.ParentChatID)
	entry.MessageID = policy.TrimText(entry.MessageID)
	entry.UpdatedAt = policy.TrimText(entry.UpdatedAt)
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
func UpsertRememberedSourceEntry(entries []RememberedSourceEntry, item RememberedSourceEntry) ([]RememberedSourceEntry, bool) {
	return policy.UpsertByNormalizedID(entries, item, NormalizeRememberedSourceEntry, func(entry RememberedSourceEntry) string {
		return entry.ID
	}, func(entry RememberedSourceEntry) bool {
		return entry.Platform != "" && entry.ID != "" && !rememberedSourceContainsControl(entry)
	})
}

func rememberedSourceContainsControl(entry RememberedSourceEntry) bool {
	return containsControl(entry.Platform) ||
		containsControl(entry.ID) ||
		containsControl(entry.Name) ||
		containsControl(entry.Type) ||
		containsControl(entry.ChatID) ||
		containsControl(entry.ChatName) ||
		containsControl(entry.UserID) ||
		containsControl(entry.UserName) ||
		containsControl(entry.ThreadID) ||
		containsControl(entry.ChatTopic) ||
		containsControl(entry.GuildID) ||
		containsControl(entry.ParentChatID) ||
		containsControl(entry.MessageID) ||
		containsControl(entry.UpdatedAt)
}

func containsControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return true
		}
	}
	return false
}
