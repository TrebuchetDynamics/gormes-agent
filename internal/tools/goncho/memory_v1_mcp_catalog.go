package gonchotools

import (
	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho/memoryv1"
)

type MemoryV1MCPToolCatalogEntry = memoryv1.MemoryV1MCPToolCatalogEntry

func RegisterMemoryV1Tools(reg *tools.Registry, store goncho.MemoryToolStore) {
	memoryv1.RegisterMemoryV1Tools(reg, store)
}

func MemoryV1MCPToolCatalog() []MemoryV1MCPToolCatalogEntry {
	return memoryv1.MemoryV1MCPToolCatalog()
}
