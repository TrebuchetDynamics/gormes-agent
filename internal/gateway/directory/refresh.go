package directory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// ChannelDirectoryAdapterSnapshot is an already-enumerated, fixture-friendly
// view of one adapter's reachable targets. Live adapter enumeration remains at
// the caller boundary so refresh logic stays hermetic and testable.
type ChannelDirectoryAdapterSnapshot struct {
	Platform string
	Entries  []ChannelDirectoryEntry
}

// ChannelDirectoryInventory lists current adapter-owned targets for a refresh.
type ChannelDirectoryInventory func(context.Context) ([]ChannelDirectoryAdapterSnapshot, error)

// ChannelDirectoryRefresher serializes channel_directory.json refreshes and
// merges adapter inventory with Manager-remembered session sources.
type ChannelDirectoryRefresher struct {
	Directory ChannelDirectoryStore
	Sources   ChannelDirectorySourceStore
	Inventory ChannelDirectoryInventory
	Now       func() time.Time

	mu sync.Mutex
}

// Refresh rebuilds the cached directory from injected adapter inventory and the
// remembered-source ledger. On refresh or write failure, the previous complete
// directory remains the returned and persisted last-known-good value.
func (r *ChannelDirectoryRefresher) Refresh(ctx context.Context) (ChannelDirectory, ChannelDirectoryEvidence) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lastGood, _ := r.Directory.Load()
	if r.Inventory == nil {
		return lastGood, ChannelDirectoryEvidence{Code: "channel_directory_refresh_failed"}
	}
	snapshots, err := r.Inventory(ctx)
	if err != nil {
		return lastGood, ChannelDirectoryEvidence{Code: "channel_directory_refresh_failed"}
	}
	dir := ChannelDirectory{UpdatedAt: refreshTimestamp(r.Now), Platforms: map[string][]ChannelDirectoryEntry{}}
	for _, snapshot := range snapshots {
		platform := strings.ToLower(strings.TrimSpace(snapshot.Platform))
		if platform == "" {
			continue
		}
		for _, entry := range snapshot.Entries {
			entry = normalizeChannelDirectoryEntry(entry)
			if entry.ID == "" || entry.Name == "" {
				continue
			}
			dir.Platforms[platform] = upsertChannelDirectoryEntry(dir.Platforms[platform], entry)
		}
	}
	ledger, sourceEvidence := r.Sources.Load()
	if sourceEvidence.Code == "" {
		for platform, entries := range ledger.Platforms {
			platform = strings.ToLower(strings.TrimSpace(platform))
			if platform == "" {
				continue
			}
			for _, source := range entries {
				entry := source.ChannelDirectoryEntry()
				entry = normalizeChannelDirectoryEntry(entry)
				if entry.ID == "" || entry.Name == "" {
					continue
				}
				dir.Platforms[platform] = upsertChannelDirectoryEntry(dir.Platforms[platform], entry)
			}
		}
	}
	sortChannelDirectory(dir)
	if err := r.Directory.Save(dir); err != nil {
		return lastGood, ChannelDirectoryEvidence{Code: "channel_directory_refresh_failed"}
	}
	return dir, ChannelDirectoryEvidence{}
}

func refreshTimestamp(now func() time.Time) string {
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(time.RFC3339)
}

func normalizeChannelDirectoryEntry(entry ChannelDirectoryEntry) ChannelDirectoryEntry {
	entry.ID = strings.TrimSpace(entry.ID)
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Type = strings.TrimSpace(entry.Type)
	entry.Guild = strings.TrimSpace(entry.Guild)
	entry.ChatID = strings.TrimSpace(entry.ChatID)
	entry.ThreadID = strings.TrimSpace(entry.ThreadID)
	entry.ChatTopic = strings.TrimSpace(entry.ChatTopic)
	return entry
}

func upsertChannelDirectoryEntry(entries []ChannelDirectoryEntry, entry ChannelDirectoryEntry) []ChannelDirectoryEntry {
	for i, existing := range entries {
		if strings.TrimSpace(existing.ID) == entry.ID {
			entries[i] = entry
			return entries
		}
	}
	return append(entries, entry)
}

func sortChannelDirectory(dir ChannelDirectory) {
	for platform, entries := range dir.Platforms {
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Name != entries[j].Name {
				return entries[i].Name < entries[j].Name
			}
			return entries[i].ID < entries[j].ID
		})
		dir.Platforms[platform] = entries
	}
}
