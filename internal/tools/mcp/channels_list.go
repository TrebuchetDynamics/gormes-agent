package mcp

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/channels"
)

// ChannelEntry represents one platform channel for MCP channels_list output.
type ChannelEntry = channels.Entry

// ChannelDirectoryProvider supplies platform channel data to MCP tools.
type ChannelDirectoryProvider = channels.DirectoryProvider

// ChannelOutput is one normalized channels_list entry.
type ChannelOutput = channels.Output

// ListChannels returns the MCP channels_list payload for a directory provider.
func ListChannels(ctx context.Context, dir ChannelDirectoryProvider, args map[string]interface{}) (interface{}, error) {
	return channels.List(ctx, dir, args)
}
