package refresh

import (
	"context"
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
		return lastGood, model.Evidence{Code: model.EvidenceChannelDirectoryRefreshFailed}
	}
	snapshots, err := r.Inventory(ctx)
	if err != nil {
		return lastGood, model.Evidence{Code: model.EvidenceChannelDirectoryRefreshFailed}
	}
	ledger, sourceEvidence := r.Sources.Load()
	dir := buildDirectory(timestamp(r.Now), snapshots, ledger, sourceEvidence.Code == "")
	if err := r.Directory.Save(dir); err != nil {
		return lastGood, model.Evidence{Code: model.EvidenceChannelDirectoryRefreshFailed}
	}
	return dir, model.Evidence{}
}

func timestamp(now func() time.Time) string {
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(time.RFC3339)
}
