package refresh

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/cache"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/sources"
)

// AdapterSnapshot is an already-enumerated, fixture-friendly view of one
// adapter's reachable targets. Live adapter enumeration remains at the caller
// boundary so refresh logic stays hermetic and testable.
type AdapterSnapshot struct {
	Platform string
	Entries  []model.Entry
}

// Inventory lists current adapter-owned targets for a refresh.
type Inventory func(context.Context) ([]AdapterSnapshot, error)

// Refresher serializes channel_directory.json refreshes and merges adapter
// inventory with Manager-remembered session sources.
type Refresher struct {
	Directory cache.Store
	Sources   sources.Store
	Inventory Inventory
	Now       func() time.Time

	mu sync.Mutex
}

// Refresh rebuilds the cached directory from injected adapter inventory and the
// remembered-source ledger. On refresh or write failure, the previous complete
// directory remains the returned and persisted last-known-good value.
func (r *Refresher) Refresh(ctx context.Context) (cache.Directory, model.Evidence) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lastGood, _ := r.Directory.Load()
	if r.Inventory == nil {
		return lastGood, model.Evidence{Code: "channel_directory_refresh_failed"}
	}
	snapshots, err := r.Inventory(ctx)
	if err != nil {
		return lastGood, model.Evidence{Code: "channel_directory_refresh_failed"}
	}
	dir := cache.NewDirectory(timestamp(r.Now))
	for _, snapshot := range snapshots {
		platform := model.NormalizePlatform(snapshot.Platform)
		if platform == "" {
			continue
		}
		for _, entry := range snapshot.Entries {
			dir.Platforms[platform], _ = model.UpsertValidEntry(dir.Platforms[platform], entry)
		}
	}
	ledger, sourceEvidence := r.Sources.Load()
	if sourceEvidence.Code == "" {
		for platform, entries := range ledger.Platforms {
			platform = model.NormalizePlatform(platform)
			if platform == "" {
				continue
			}
			for _, source := range entries {
				dir.Platforms[platform], _ = model.UpsertValidEntry(dir.Platforms[platform], source.ChannelDirectoryEntry())
			}
		}
	}
	sortDirectory(dir)
	if err := r.Directory.Save(dir); err != nil {
		return lastGood, model.Evidence{Code: "channel_directory_refresh_failed"}
	}
	return dir, model.Evidence{}
}

func timestamp(now func() time.Time) string {
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(time.RFC3339)
}

func sortDirectory(dir cache.Directory) {
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
