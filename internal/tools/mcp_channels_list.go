package tools

import (
	"context"

	mcppkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
)

// ChannelEntry represents one platform channel for MCP channels_list output.
type ChannelEntry = mcppkg.ChannelEntry

// ChannelDirectoryProvider supplies platform channel data to MCP tools.
type ChannelDirectoryProvider = mcppkg.ChannelDirectoryProvider

type channelOutput = mcppkg.ChannelOutput

func (s *MCPServer) channelsListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return mcppkg.ListChannels(ctx, s.channelDir, args)
}
