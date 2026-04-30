package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const channelDirectorySourcesFileName = "channel_directory_sources.json"

// RememberedSourceStore is the fakeable ledger seam used by Manager to persist
// allowed inbound channel sources without mutating channel_directory.json. A
// later refresh slice can merge this ledger into the directory read model.
type RememberedSourceStore interface {
	RememberSource(context.Context, RememberedSourceEntry) error
}

// RememberedSourceEntry is the session-origin-shaped source record preserved
// for channel-directory refresh. Fields intentionally mirror Hermes session
// origin data plus enough metadata to produce ChannelDirectoryEntry values.
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

// ChannelDirectorySourceStore persists a remembered-source ledger under a
// caller-owned root. It is distinct from channel_directory.json on purpose.
type ChannelDirectorySourceStore struct {
	root string
	now  func() time.Time
}

func NewChannelDirectorySourceStore(root string) ChannelDirectorySourceStore {
	return ChannelDirectorySourceStore{root: strings.TrimSpace(root), now: time.Now}
}

func (s ChannelDirectorySourceStore) path() string {
	return filepath.Join(s.root, channelDirectorySourcesFileName)
}

func (s ChannelDirectorySourceStore) RememberSource(_ context.Context, entry RememberedSourceEntry) error {
	if strings.TrimSpace(s.root) == "" {
		return fmt.Errorf("channel directory source root is empty")
	}
	entry = normalizeRememberedSourceEntry(entry)
	if entry.Platform == "" || entry.ID == "" {
		return nil
	}
	ledger := RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}
	if body, err := os.ReadFile(s.path()); err == nil && len(body) > 0 {
		_ = json.Unmarshal(body, &ledger)
	}
	if ledger.Platforms == nil {
		ledger.Platforms = map[string][]RememberedSourceEntry{}
	}
	if s.now == nil {
		s.now = time.Now
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	entry.UpdatedAt = now
	ledger.UpdatedAt = now
	entries := ledger.Platforms[entry.Platform]
	for i, existing := range entries {
		if strings.TrimSpace(existing.ID) == entry.ID {
			entries[i] = entry
			ledger.Platforms[entry.Platform] = entries
			return s.save(ledger)
		}
	}
	ledger.Platforms[entry.Platform] = append(entries, entry)
	return s.save(ledger)
}

func (s ChannelDirectorySourceStore) save(ledger RememberedSourceLedger) error {
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	tmp, err := os.CreateTemp(s.root, ".channel_directory_sources-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, s.path()); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func RememberedSourceEntryFromSessionSource(source SessionSource) RememberedSourceEntry {
	entry := RememberedSourceEntry{
		Platform:     strings.ToLower(strings.TrimSpace(source.Platform)),
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
	return normalizeRememberedSourceEntry(entry)
}

func (e RememberedSourceEntry) ChannelDirectoryEntry() ChannelDirectoryEntry {
	return ChannelDirectoryEntry{
		ID:        e.ID,
		Name:      e.Name,
		Type:      e.Type,
		Guild:     e.GuildID,
		ChatID:    e.ChatID,
		ThreadID:  e.ThreadID,
		ChatTopic: e.ChatTopic,
	}
}

func normalizeRememberedSourceEntry(entry RememberedSourceEntry) RememberedSourceEntry {
	entry.Platform = strings.ToLower(strings.TrimSpace(entry.Platform))
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

func normalizedSourceChatType(source SessionSource) string {
	if chatType := strings.TrimSpace(source.ChatType); chatType != "" {
		return chatType
	}
	if strings.TrimSpace(source.ThreadID) != "" {
		return "thread"
	}
	return "dm"
}
