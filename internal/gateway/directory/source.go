package directory

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/sources"
)

// Source is kept as the package-level compatibility name for the shared
// session-origin-shaped source value contract.
type Source = model.Source

// RememberedSourceEntry is kept as the package-level compatibility name for the
// shared remembered-source value contract.
type RememberedSourceEntry = model.RememberedSourceEntry

// RememberedSourceLedger is kept as the package-level compatibility name for
// the shared remembered-source ledger value contract.
type RememberedSourceLedger = model.RememberedSourceLedger

// RememberedSourceStore is the fakeable ledger seam used by Manager to persist
// allowed inbound channel sources without mutating channel_directory.json. A
// later refresh slice can merge this ledger into the directory read model.
type RememberedSourceStore interface {
	RememberSource(context.Context, RememberedSourceEntry) error
}

// ChannelDirectorySourceStore persists a remembered-source ledger under a
// caller-owned root. It is distinct from channel_directory.json on purpose.
type ChannelDirectorySourceStore = sources.Store

func NewChannelDirectorySourceStore(root string) ChannelDirectorySourceStore {
	return sources.NewStore(root)
}

func RememberedSourceEntryFromSource(source Source) RememberedSourceEntry {
	return model.RememberedSourceEntryFromSource(source)
}
