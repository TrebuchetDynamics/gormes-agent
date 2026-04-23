package gateway

import (
	"sort"
	"strings"
	"sync"
)

// ChannelDirectoryEntry is the cached, platform-neutral name metadata for a
// chat/contact observed by the gateway. It is intentionally persistence-ready,
// but ChannelDirectory itself is in-memory for this phase slice.
type ChannelDirectoryEntry struct {
	Platform string `json:"platform"`
	ChatID   string `json:"chat_id"`
	ChatName string `json:"chat_name,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	UserName string `json:"user_name,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`
}

// ChannelDirectory records known chat/contact metadata and resolves either
// stable chat IDs or human-readable chat names back to the latest entry.
type ChannelDirectory struct {
	mu              sync.RWMutex
	entries         map[string]ChannelDirectoryEntry
	chatNameAliases map[string]string
}

// NewChannelDirectory returns an empty channel/contact directory.
func NewChannelDirectory() *ChannelDirectory {
	return &ChannelDirectory{
		entries:         map[string]ChannelDirectoryEntry{},
		chatNameAliases: map[string]string{},
	}
}

// UpdateFromInbound records the routing metadata carried by an inbound event.
// Empty platform or chat IDs are ignored; empty names preserve existing cache
// values so sparse adapter events do not erase useful aliases.
func (d *ChannelDirectory) UpdateFromInbound(ev InboundEvent) (ChannelDirectoryEntry, bool) {
	if d == nil {
		return ChannelDirectoryEntry{}, false
	}

	platform := normalizePlatform(ev.Platform)
	chatID := strings.TrimSpace(ev.ChatID)
	if platform == "" || chatID == "" {
		return ChannelDirectoryEntry{}, false
	}

	key := directoryKey(platform, chatID)
	next := ChannelDirectoryEntry{
		Platform: platform,
		ChatID:   chatID,
		ChatName: strings.TrimSpace(ev.ChatName),
		UserID:   strings.TrimSpace(ev.UserID),
		UserName: strings.TrimSpace(ev.UserName),
		ThreadID: strings.TrimSpace(ev.ThreadID),
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	prev, hadPrev := d.entries[key]
	if hadPrev {
		next = mergeDirectoryEntry(prev, next)
		d.dropRenamedAlias(platform, key, prev.ChatName, next.ChatName)
	}

	d.entries[key] = next
	if next.ChatName != "" {
		d.chatNameAliases[aliasKey(platform, next.ChatName)] = key
	}
	return next, true
}

// Lookup returns a directory entry by platform plus either chat ID or chat
// name. Chat IDs are matched after trimming; names are case-insensitive.
func (d *ChannelDirectory) Lookup(platform, chatIDOrName string) (ChannelDirectoryEntry, bool) {
	if d == nil {
		return ChannelDirectoryEntry{}, false
	}

	platform = normalizePlatform(platform)
	target := strings.TrimSpace(chatIDOrName)
	if platform == "" || target == "" {
		return ChannelDirectoryEntry{}, false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	if entry, ok := d.entries[directoryKey(platform, target)]; ok {
		return entry, true
	}
	key, ok := d.chatNameAliases[aliasKey(platform, target)]
	if !ok {
		return ChannelDirectoryEntry{}, false
	}
	entry, ok := d.entries[key]
	return entry, ok
}

// Snapshot returns a deterministic copy of all known entries for future
// persistence and operator status surfaces.
func (d *ChannelDirectory) Snapshot() []ChannelDirectoryEntry {
	if d == nil {
		return nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make([]ChannelDirectoryEntry, 0, len(d.entries))
	for _, entry := range d.entries {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return out[i].ChatID < out[j].ChatID
	})
	return out
}

func normalizePlatform(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func mergeDirectoryEntry(prev, next ChannelDirectoryEntry) ChannelDirectoryEntry {
	if next.ChatName == "" {
		next.ChatName = prev.ChatName
	}
	if next.UserID == "" {
		next.UserID = prev.UserID
	}
	if next.UserName == "" {
		next.UserName = prev.UserName
	}
	if next.ThreadID == "" {
		next.ThreadID = prev.ThreadID
	}
	return next
}

func (d *ChannelDirectory) dropRenamedAlias(platform, oldKey, oldName, newName string) {
	if oldName != "" && !strings.EqualFold(oldName, newName) {
		key := aliasKey(platform, oldName)
		if d.chatNameAliases[key] == oldKey {
			delete(d.chatNameAliases, key)
		}
	}
}

func directoryKey(platform, chatID string) string {
	return platform + "\x00" + chatID
}

func aliasKey(platform, name string) string {
	return platform + "\x00" + strings.ToLower(strings.TrimSpace(name))
}
