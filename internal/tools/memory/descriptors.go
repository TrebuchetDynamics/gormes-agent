package memory

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/memory/catalog"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const (
	StoreMemoryToolName       = catalog.StoreMemoryToolName
	RetrieveMemoryToolName    = catalog.RetrieveMemoryToolName
	UpdateMemoryToolName      = catalog.UpdateMemoryToolName
	SummarizeMemoriesToolName = catalog.SummarizeMemoriesToolName
	ForgetMemoryToolName      = catalog.ForgetMemoryToolName
)

// MemoryToolDescriptors returns the MCP-compatible agent memory tool catalog in
// the same stable order an agent usually chains the operations.
func MemoryToolDescriptors() []toolkit.ToolDescriptor { return catalog.MemoryToolDescriptors() }

// MemoryToolOperationSpecs returns behavioral metadata for the agent-callable
// memory tools.
func MemoryToolOperationSpecs() []toolkit.OperationSpec { return catalog.MemoryToolOperationSpecs() }

// MemoryToolOperationSpec returns the behavioral metadata for one memory tool.
func MemoryToolOperationSpec(name string) (toolkit.OperationSpec, bool) {
	return catalog.MemoryToolOperationSpec(name)
}
