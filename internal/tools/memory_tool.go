package tools

import toolmemory "github.com/TrebuchetDynamics/gormes-agent/internal/tools/memory"

const (
	MemoryToolName = toolmemory.MemoryToolName

	MemoryEvidenceInvalidArgs      = toolmemory.MemoryEvidenceInvalidArgs
	MemoryEvidenceStoreUnavailable = toolmemory.MemoryEvidenceStoreUnavailable
	MemoryEvidenceUnsafeContent    = toolmemory.MemoryEvidenceUnsafeContent
	MemoryEvidenceEntryNotFound    = toolmemory.MemoryEvidenceEntryNotFound
	MemoryEvidenceAmbiguousMatch   = toolmemory.MemoryEvidenceAmbiguousMatch
	MemoryEvidenceLimitExceeded    = toolmemory.MemoryEvidenceLimitExceeded
)

type MemoryToolConfig = toolmemory.MemoryToolConfig
type MemoryTool = toolmemory.MemoryTool
type MemoryToolResult = toolmemory.MemoryToolResult
type MemoryInventoryProvenance = toolmemory.MemoryInventoryProvenance
type MemoryInventorySource = toolmemory.MemoryInventorySource

func NewMemoryTool(cfg MemoryToolConfig) *MemoryTool { return toolmemory.NewMemoryTool(cfg) }
