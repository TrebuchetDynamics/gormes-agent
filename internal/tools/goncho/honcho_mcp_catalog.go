package gonchotools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho/catalog"

// HonchoMCPToolStatus records whether an upstream Honcho MCP tool is covered
// by an existing Goncho tool descriptor or remains an explicit compatibility
// gap.
type HonchoMCPToolStatus = catalog.HonchoMCPToolStatus

const (
	HonchoMCPToolMapped      = catalog.HonchoMCPToolMapped
	HonchoMCPToolPartial     = catalog.HonchoMCPToolPartial
	HonchoMCPToolUnsupported = catalog.HonchoMCPToolUnsupported
)

// HonchoMCPToolCatalogEntry is a source-backed compatibility row for one tool
// registered by ../honcho/mcp/src/tools/*.ts.
type HonchoMCPToolCatalogEntry = catalog.HonchoMCPToolCatalogEntry

// HonchoMCPToolCatalog returns the upstream Honcho MCP tool matrix in the same
// module order used by mcp/src/server.ts: workspace, peers, sessions,
// conclusions, then system.
func HonchoMCPToolCatalog() []HonchoMCPToolCatalogEntry {
	return catalog.HonchoMCPToolCatalog()
}
