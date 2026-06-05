package gateway

import gatewaydirectory "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory"

// RememberedSourceStore is the fakeable ledger seam used by Manager to persist
// allowed inbound channel sources without mutating channel_directory.json. A
// later refresh slice can merge this ledger into the directory read model.
type RememberedSourceStore = gatewaydirectory.RememberedSourceStore

// RememberedSourceEntry is the session-origin-shaped source record preserved
// for channel-directory refresh. Fields intentionally mirror Hermes session
// origin data plus enough metadata to produce ChannelDirectoryEntry values.
type RememberedSourceEntry = gatewaydirectory.RememberedSourceEntry

// RememberedSourceLedger is the on-disk remembered-source ledger shape.
type RememberedSourceLedger = gatewaydirectory.RememberedSourceLedger

// ChannelDirectorySourceStore persists a remembered-source ledger under a
// caller-owned root. It is distinct from channel_directory.json on purpose.
type ChannelDirectorySourceStore = gatewaydirectory.ChannelDirectorySourceStore

func NewChannelDirectorySourceStore(root string) ChannelDirectorySourceStore {
	return gatewaydirectory.NewChannelDirectorySourceStore(root)
}

func RememberedSourceEntryFromSessionSource(source SessionSource) RememberedSourceEntry {
	return gatewaydirectory.RememberedSourceEntryFromSource(gatewaydirectory.Source{
		Platform:     source.Platform,
		ChatID:       source.ChatID,
		ChatName:     source.ChatName,
		ChatType:     source.ChatType,
		UserID:       source.UserID,
		UserName:     source.UserName,
		ThreadID:     source.ThreadID,
		GuildID:      source.GuildID,
		ParentChatID: source.ParentChatID,
		MessageID:    source.MessageID,
	})
}
