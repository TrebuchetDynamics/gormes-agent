package directory

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/cache"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

// ChannelDirectory is the channel-neutral cached read model for reachable
// messaging targets. It mirrors Hermes' channel_directory.json shape while
// keeping Gormes runtime behavior native Go and fixture-driven.
type ChannelDirectory = cache.Directory

// ChannelDirectoryEntry is kept as the package-level compatibility name for
// the shared directory target value contract.
type ChannelDirectoryEntry = model.Entry

// ChannelDirectoryEvidence is kept as the package-level compatibility name for
// shared user-safe degraded-mode evidence.
type ChannelDirectoryEvidence = model.Evidence

// ChannelDirectoryStore persists channel_directory.json under a caller-owned
// Gormes state/home root. Tests pass temp roots; no live operator home is read.
type ChannelDirectoryStore = cache.Store

// NewChannelDirectoryStore returns a store rooted at root.
func NewChannelDirectoryStore(root string) ChannelDirectoryStore {
	return cache.NewStore(root)
}
