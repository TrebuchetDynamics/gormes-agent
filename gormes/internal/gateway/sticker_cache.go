package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// StickerCacheRecord is the generic, persistence-ready sticker lookup surface
// that future adapters can bind to platform-native sticker identifiers.
type StickerCacheRecord struct {
	Platform  string    `json:"platform"`
	ChatID    string    `json:"chat_id,omitempty"`
	Alias     string    `json:"alias"`
	StickerID string    `json:"sticker_id"`
	CachedAt  time.Time `json:"cached_at"`
}

// StickerCache persists one normalized sticker lookup cache for future
// adapter-specific sticker send/resolve flows.
type StickerCache struct {
	mu      sync.RWMutex
	path    string
	entries map[string]StickerCacheRecord
}

func OpenStickerCache(path string) (*StickerCache, error) {
	cache := &StickerCache{
		path:    path,
		entries: map[string]StickerCacheRecord{},
	}
	if strings.TrimSpace(path) == "" {
		return cache, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, fmt.Errorf("gateway: read sticker cache %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return cache, nil
	}

	var records []StickerCacheRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("gateway: decode sticker cache %s: %w", path, err)
	}
	for _, record := range records {
		record = normalizeStickerCacheRecord(record)
		if record.Platform == "" || record.Alias == "" || record.StickerID == "" {
			continue
		}
		cache.entries[stickerCacheKey(record.Platform, record.ChatID, record.Alias)] = record
	}
	return cache, nil
}

func (c *StickerCache) Path() string {
	if c == nil {
		return ""
	}
	return c.path
}

func (c *StickerCache) Put(record StickerCacheRecord) error {
	if c == nil {
		return nil
	}
	record = normalizeStickerCacheRecord(record)
	if record.Platform == "" || record.Alias == "" || record.StickerID == "" {
		return fmt.Errorf("gateway: platform, alias, and sticker_id are required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[stickerCacheKey(record.Platform, record.ChatID, record.Alias)] = record
	return c.persistLocked()
}

func (c *StickerCache) Lookup(platform, chatID, alias string) (StickerCacheRecord, bool) {
	if c == nil {
		return StickerCacheRecord{}, false
	}

	key := stickerCacheKey(platform, chatID, alias)
	if key == "" {
		return StickerCacheRecord{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	record, ok := c.entries[key]
	return record, ok
}

func (c *StickerCache) Snapshot() []StickerCacheRecord {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return snapshotStickerCacheEntries(c.entries)
}

func snapshotStickerCacheEntries(entries map[string]StickerCacheRecord) []StickerCacheRecord {
	out := make([]StickerCacheRecord, 0, len(entries))
	for _, record := range entries {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		if out[i].ChatID != out[j].ChatID {
			return out[i].ChatID < out[j].ChatID
		}
		return strings.ToLower(out[i].Alias) < strings.ToLower(out[j].Alias)
	})
	return out
}

func (c *StickerCache) persistLocked() error {
	if strings.TrimSpace(c.path) == "" {
		return nil
	}

	data, err := json.MarshalIndent(snapshotStickerCacheEntries(c.entries), "", "  ")
	if err != nil {
		return fmt.Errorf("gateway: encode sticker cache %s: %w", c.path, err)
	}
	return writeJSONAtomic(c.path, data, 0o600)
}

func normalizeStickerCacheRecord(record StickerCacheRecord) StickerCacheRecord {
	record.Platform = normalizePlatform(record.Platform)
	record.ChatID = strings.TrimSpace(record.ChatID)
	record.Alias = strings.TrimSpace(record.Alias)
	record.StickerID = strings.TrimSpace(record.StickerID)
	return record
}

func stickerCacheKey(platform, chatID, alias string) string {
	platform = normalizePlatform(platform)
	alias = strings.ToLower(strings.TrimSpace(alias))
	chatID = strings.TrimSpace(chatID)
	if platform == "" || alias == "" {
		return ""
	}
	return platform + "\x00" + chatID + "\x00" + alias
}
