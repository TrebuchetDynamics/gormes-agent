package gonchotools

import (
	"sort"

	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type MemoryV1MCPToolCatalogEntry struct {
	Name            string `json:"name"`
	ContractVersion string `json:"contract_version"`
	LocalFirst      bool   `json:"local_first"`
	MarkdownBacked  bool   `json:"markdown_backed"`
	RequiresNetwork bool   `json:"requires_network"`
	RequiresOllama  bool   `json:"requires_ollama"`
	Source          string `json:"source"`
}

func RegisterMemoryV1Tools(reg *tools.Registry, store goncho.MemoryToolStore) {
	if reg == nil {
		panic("tools: nil registry")
	}
	if store == nil {
		panic("tools: nil goncho memory store")
	}
	reg.MustRegister(goncho.NewStoreMemoryTool(store))
	reg.MustRegister(goncho.NewRetrieveMemoryTool(store))
	reg.MustRegister(goncho.NewUpdateMemoryTool(store))
	reg.MustRegister(goncho.NewSummarizeMemoryTool(store))
	reg.MustRegister(goncho.NewForgetMemoryTool(store))
}

func MemoryV1MCPToolCatalog() []MemoryV1MCPToolCatalogEntry {
	contract := goncho.MemoryV1ToolContract()
	names := make([]string, 0, len(contract.Tools))
	for name := range contract.Tools {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]MemoryV1MCPToolCatalogEntry, 0, len(names))
	for _, name := range names {
		out = append(out, MemoryV1MCPToolCatalogEntry{
			Name:            name,
			ContractVersion: contract.ContractVersion,
			LocalFirst:      true,
			MarkdownBacked:  true,
			RequiresNetwork: false,
			RequiresOllama:  false,
			Source:          "goncho_memory_v1",
		})
	}
	return out
}
