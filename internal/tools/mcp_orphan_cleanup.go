package tools

import mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"

type MCPStdioCleanupStatus = mcptools.MCPStdioCleanupStatus

const (
	MCPOrphanReaped       = mcptools.MCPOrphanReaped
	MCPOrphanReapFailed   = mcptools.MCPOrphanReapFailed
	MCPActivePIDPreserved = mcptools.MCPActivePIDPreserved
)

type MCPStdioCleanupEvent = mcptools.MCPStdioCleanupEvent
type MCPStdioProcessTrackerOptions = mcptools.MCPStdioProcessTrackerOptions
type MCPStdioProcessSnapshot = mcptools.MCPStdioProcessSnapshot
type MCPStdioProcessTracker = mcptools.MCPStdioProcessTracker

var DefaultMCPStdioProcessTracker = mcptools.DefaultMCPStdioProcessTracker

func NewMCPStdioProcessTracker(opts MCPStdioProcessTrackerOptions) *MCPStdioProcessTracker {
	return mcptools.NewMCPStdioProcessTracker(opts)
}

func ReapMCPStdioOrphans() []MCPStdioCleanupEvent {
	return DefaultMCPStdioProcessTracker.ReapOrphans()
}

func ShutdownMCPStdioProcesses() []MCPStdioCleanupEvent {
	return DefaultMCPStdioProcessTracker.Shutdown()
}
