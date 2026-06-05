package gateway

import gatewaydirectory "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory"

// ChannelDirectory is the channel-neutral cached read model for reachable
// messaging targets. It mirrors Hermes' channel_directory.json shape while
// keeping Gormes runtime behavior native Go and fixture-driven.
type ChannelDirectory = gatewaydirectory.ChannelDirectory

// ChannelDirectoryEntry describes one platform target that can be selected by
// exact ID, human display name, guild-qualified name, or type/display lookup.
type ChannelDirectoryEntry = gatewaydirectory.ChannelDirectoryEntry

// ChannelDirectoryEvidence carries user-safe degraded-mode evidence without
// leaking local state paths.
type ChannelDirectoryEvidence = gatewaydirectory.ChannelDirectoryEvidence

// ChannelDirectoryStore persists channel_directory.json under a caller-owned
// Gormes state/home root. Tests pass temp roots; no live operator home is read.
type ChannelDirectoryStore = gatewaydirectory.ChannelDirectoryStore

// NewChannelDirectoryStore returns a store rooted at root.
func NewChannelDirectoryStore(root string) ChannelDirectoryStore {
	return gatewaydirectory.NewChannelDirectoryStore(root)
}
