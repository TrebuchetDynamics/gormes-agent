package gateway

import gatewaydirectory "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory"

// ChannelDirectoryAdapterSnapshot is an already-enumerated, fixture-friendly
// view of one adapter's reachable targets. Live adapter enumeration remains at
// the caller boundary so refresh logic stays hermetic and testable.
type ChannelDirectoryAdapterSnapshot = gatewaydirectory.ChannelDirectoryAdapterSnapshot

// ChannelDirectoryInventory lists current adapter-owned targets for a refresh.
type ChannelDirectoryInventory = gatewaydirectory.ChannelDirectoryInventory

// ChannelDirectoryRefresher serializes channel_directory.json refreshes and
// merges adapter inventory with Manager-remembered session sources.
type ChannelDirectoryRefresher = gatewaydirectory.ChannelDirectoryRefresher
