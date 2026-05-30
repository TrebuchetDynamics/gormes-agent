package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/memory"

const (
	StoreMemoryToolName       = memory.StoreMemoryToolName
	RetrieveMemoryToolName    = memory.RetrieveMemoryToolName
	UpdateMemoryToolName      = memory.UpdateMemoryToolName
	SummarizeMemoriesToolName = memory.SummarizeMemoriesToolName
	ForgetMemoryToolName      = memory.ForgetMemoryToolName
)

func MemoryToolDescriptors() []ToolDescriptor {
	return memory.MemoryToolDescriptors()
}

func MemoryToolOperationSpecs() []OperationSpec {
	return memory.MemoryToolOperationSpecs()
}

func MemoryToolOperationSpec(name string) (OperationSpec, bool) {
	return memory.MemoryToolOperationSpec(name)
}
