package channels

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/channels/listing"
)

// Entry represents one platform channel for MCP channels_list output.
type Entry = listing.Entry

// DirectoryProvider supplies platform channel data to MCP tools.
type DirectoryProvider = listing.DirectoryProvider

// Output is one normalized channels_list entry.
type Output = listing.Output

// List returns the MCP channels_list payload for a directory provider.
func List(ctx context.Context, dir DirectoryProvider, args map[string]interface{}) (interface{}, error) {
	return listing.List(ctx, dir, args)
}
