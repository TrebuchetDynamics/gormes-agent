package mcp

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/process"

// MCPStdioCleanupStatus is operator-visible evidence from MCP stdio cleanup.
type MCPStdioCleanupStatus = process.MCPStdioCleanupStatus

const (
	MCPOrphanReaped       MCPStdioCleanupStatus = process.MCPOrphanReaped
	MCPOrphanReapFailed   MCPStdioCleanupStatus = process.MCPOrphanReapFailed
	MCPActivePIDPreserved MCPStdioCleanupStatus = process.MCPActivePIDPreserved
)

// MCPStdioCleanupEvent records one PID decision from a cleanup sweep.
type MCPStdioCleanupEvent = process.MCPStdioCleanupEvent

// MCPStdioProcessTrackerOptions injects process operations for tests.
type MCPStdioProcessTrackerOptions = process.MCPStdioProcessTrackerOptions

// MCPStdioProcessSnapshot is a copied read model of tracked PIDs.
type MCPStdioProcessSnapshot = process.MCPStdioProcessSnapshot

// MCPStdioProcessTracker tracks active stdio MCP server PIDs and the subset
// that survived session exit.
type MCPStdioProcessTracker = process.MCPStdioProcessTracker

// DefaultMCPStdioProcessTracker is used by stdio clients and cron cleanup in
// normal runtime wiring.
var DefaultMCPStdioProcessTracker = process.DefaultMCPStdioProcessTracker

func NewMCPStdioProcessTracker(opts MCPStdioProcessTrackerOptions) *MCPStdioProcessTracker {
	return process.NewMCPStdioProcessTracker(opts)
}

// ReapMCPStdioOrphans runs the normal post-cron cleanup sweep.
func ReapMCPStdioOrphans() []MCPStdioCleanupEvent {
	return process.ReapMCPStdioOrphans()
}

// ShutdownMCPStdioProcesses runs the final shutdown sweep, including active
// PIDs because no stdio sessions should remain in flight.
func ShutdownMCPStdioProcesses() []MCPStdioCleanupEvent {
	return process.ShutdownMCPStdioProcesses()
}
