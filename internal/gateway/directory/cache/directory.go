package cache

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/cache/readmodel"

// Directory is the channel-neutral cached read model for reachable messaging
// targets. It mirrors Hermes' channel_directory.json shape while keeping Gormes
// runtime behavior native Go and fixture-driven.
type Directory = readmodel.Directory

// Store persists channel_directory.json under a caller-owned Gormes state/home
// root. Tests pass temp roots; no live operator home is read.
type Store = readmodel.Store

// NewStore returns a store rooted at root.
func NewStore(root string) Store {
	return readmodel.NewStore(root)
}

// NewDirectory returns a directory with initialized platform buckets.
func NewDirectory(updatedAt string) Directory {
	return readmodel.NewDirectory(updatedAt)
}
