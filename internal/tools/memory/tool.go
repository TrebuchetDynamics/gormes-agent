package memory

import durable "github.com/TrebuchetDynamics/gormes-agent/internal/tools/memory/durable"

const (
	MemoryToolName = durable.MemoryToolName

	MemoryEvidenceInvalidArgs      = durable.MemoryEvidenceInvalidArgs
	MemoryEvidenceStoreUnavailable = durable.MemoryEvidenceStoreUnavailable
	MemoryEvidenceUnsafeContent    = durable.MemoryEvidenceUnsafeContent
	MemoryEvidenceEntryNotFound    = durable.MemoryEvidenceEntryNotFound
	MemoryEvidenceAmbiguousMatch   = durable.MemoryEvidenceAmbiguousMatch
	MemoryEvidenceLimitExceeded    = durable.MemoryEvidenceLimitExceeded
)

type MemoryToolConfig = durable.MemoryToolConfig
type MemoryTool = durable.MemoryTool
type MemoryToolResult = durable.MemoryToolResult
type MemoryInventoryProvenance = durable.MemoryInventoryProvenance
type MemoryInventorySource = durable.MemoryInventorySource

func NewMemoryTool(cfg MemoryToolConfig) *MemoryTool { return durable.NewMemoryTool(cfg) }
