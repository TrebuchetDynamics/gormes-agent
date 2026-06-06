package memoryv1

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestMemoryV1MCPCatalogRegistersLocalMarkdownTools(t *testing.T) {
	ctx := context.Background()
	sqlite, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer sqlite.Close(ctx)
	store := goncho.NewLocalMarkdownMemoryStore(sqlite.DB(), goncho.LocalMarkdownMemoryConfig{
		Path:        filepath.Join(t.TempDir(), "GONCHO_MEMORY.md"),
		AgentID:     "agent-a",
		WorkspaceID: "workspace-private",
		PeerID:      "user-juan",
	})

	reg := tools.NewRegistry()
	RegisterMemoryV1Tools(reg, store)
	contract := goncho.MemoryV1ToolContract()
	for name := range contract.Tools {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if len(tool.Schema()) == 0 || tool.Description() == "" {
			t.Fatalf("%s descriptor missing schema/description", name)
		}
	}
	if _, ok := reg.Get("purge_memory"); ok {
		t.Fatal("purge_memory registered in normal MCP tool set")
	}

	catalog := MemoryV1MCPToolCatalog()
	if len(catalog) != len(contract.Tools) {
		t.Fatalf("catalog entries = %d, want %d", len(catalog), len(contract.Tools))
	}
	for _, entry := range catalog {
		if !entry.LocalFirst || !entry.MarkdownBacked || entry.RequiresNetwork || entry.RequiresOllama {
			t.Fatalf("catalog entry = %+v, want local markdown tool with optional Ollama", entry)
		}
		if entry.ContractVersion != contract.ContractVersion {
			t.Fatalf("%s contract = %q, want %q", entry.Name, entry.ContractVersion, contract.ContractVersion)
		}
	}
}

func TestMemoryV1MCPToolsReturnContractedLocalFirstResults(t *testing.T) {
	ctx := context.Background()
	sqlite, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer sqlite.Close(ctx)

	reg := tools.NewRegistry()
	RegisterMemoryV1Tools(reg, goncho.NewLocalMarkdownMemoryStore(sqlite.DB(), goncho.LocalMarkdownMemoryConfig{
		Path:        filepath.Join(t.TempDir(), "GONCHO_MEMORY.md"),
		AgentID:     "agent-a",
		WorkspaceID: "workspace-private",
		PeerID:      "user-juan",
	}))

	stored := executeMemoryV1Tool(t, reg, "store_memory", json.RawMessage(`{
		"content":"Goncho local markdown MCP memory works without API keys.",
		"tags":["local","mcp"],
		"importance":0.9
	}`))
	if stored["contract_version"] != "1" || stored["local_first"] != true || stored["markdown_backed"] != true {
		t.Fatalf("store_memory result = %+v, want V1 local markdown contract", stored)
	}

	retrieved := executeMemoryV1Tool(t, reg, "retrieve_memory", json.RawMessage(`{
		"query":"API keys",
		"limit":5
	}`))
	if retrieved["contract_version"] != "1" || retrieved["local_first"] != true {
		t.Fatalf("retrieve_memory result = %+v, want V1 local contract", retrieved)
	}
	rawResults, ok := retrieved["results"].([]any)
	if !ok || len(rawResults) != 1 {
		t.Fatalf("retrieve results = %#v, want one result", retrieved["results"])
	}
	if !strings.Contains(string(mustJSON(t, rawResults[0])), "without API keys") {
		t.Fatalf("retrieve output = %+v, want stored memory", retrieved)
	}
}

func executeMemoryV1Tool(t *testing.T, reg *tools.Registry, name string, input json.RawMessage) map[string]any {
	t.Helper()
	ch, err := tools.NewInProcessToolExecutor(reg).Execute(context.Background(), tools.ToolRequest{
		ToolName: name,
		Input:    input,
	})
	if err != nil {
		t.Fatal(err)
	}
	var outputs []tools.ToolEvent
	for ev := range ch {
		outputs = append(outputs, ev)
	}
	if len(outputs) != 3 || outputs[1].Type != "output" {
		t.Fatalf("outputs = %+v, want start/output/end", outputs)
	}
	var out map[string]any
	if err := json.Unmarshal(outputs[1].Output, &out); err != nil {
		t.Fatalf("decode %s output %s: %v", name, outputs[1].Output, err)
	}
	return out
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestMemoryV1MCPCatalogNamesStayStable(t *testing.T) {
	names := make([]string, 0, len(MemoryV1MCPToolCatalog()))
	for _, entry := range MemoryV1MCPToolCatalog() {
		names = append(names, entry.Name)
	}
	slices.Sort(names)
	want := []string{"forget_memory", "retrieve_memory", "store_memory", "summarize_memories", "update_memory"}
	if !slices.Equal(names, want) {
		t.Fatalf("catalog names = %#v, want %#v", names, want)
	}
}
