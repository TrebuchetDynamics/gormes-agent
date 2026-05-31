package refresh

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/cache"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

// buildDirectory is the refresh package's shared merge contract: adapter-owned
// inventory is always merged, remembered-source ledger entries are merged only
// when their ledger decoded cleanly, and persisted entries are sorted before
// the cache store sees them.
func buildDirectory(updatedAt string, snapshots []AdapterSnapshot, ledger model.RememberedSourceLedger, includeLedger bool) cache.Directory {
	dir := cache.NewDirectory(updatedAt)
	mergeSnapshots(&dir, snapshots)
	if includeLedger {
		mergeRememberedSources(&dir, ledger)
	}
	sortDirectory(dir)
	return dir
}

func mergeSnapshots(dir *cache.Directory, snapshots []AdapterSnapshot) {
	for _, snapshot := range snapshots {
		dir.UpsertEntries(snapshot.Platform, snapshot.Entries...)
	}
}

func mergeRememberedSources(dir *cache.Directory, ledger model.RememberedSourceLedger) {
	for platform, entries := range ledger.Platforms {
		for _, source := range entries {
			dir.UpsertEntries(platform, source.ChannelDirectoryEntry())
		}
	}
}

func sortDirectory(dir cache.Directory) {
	for platform, entries := range dir.Platforms {
		model.SortEntriesByNameID(entries)
		dir.Platforms[platform] = entries
	}
}
