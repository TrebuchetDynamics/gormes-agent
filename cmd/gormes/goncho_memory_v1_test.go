package main

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestGonchoDoctorCommand_TextReportsMemoryV1Contract(t *testing.T) {
	seedGonchoDoctorZeroStateDB(t)

	stdout, stderr, err := runGonchoDoctorCommand(t, "goncho", "doctor")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Memory contract",
		"contract_version: 1",
		"markdown_format_version: 1",
		"mcp_tool_contract_version: 1",
		"private_agent_memory_default: true",
		"self_improvement_per_agent_default: true",
		"foreign_config_runtime_reads: denied",
		"fast_recall_path: sqlite, fts5, graph",
		"optional_quality_layers: embeddings, qmd_deep_search, dialectic, dream_consolidation",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestGonchoDoctorCommandReportsLocalMarkdownMCPMemory(t *testing.T) {
	seedGonchoDoctorZeroStateDB(t)

	stdout, stderr, err := runGonchoDoctorCommand(t, "goncho", "doctor")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Local markdown memory",
		"path: " + filepath.Join(config.GormesHome(), "memory", "GONCHO_MEMORY.md"),
		"enabled: true",
		"sqlite_backed: true",
		"markdown_backed: true",
		"network_required: false",
		"ollama_required: false",
		"mcp_tools: forget_memory, retrieve_memory, store_memory, summarize_memories, update_memory",
		"store_memory",
		"retrieve_memory",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestGonchoDoctorCommandJSONReportsLocalMarkdownMCPMemory(t *testing.T) {
	seedGonchoDoctorZeroStateDB(t)

	stdout, stderr, err := runGonchoDoctorCommand(t, "goncho", "doctor", "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		LocalMarkdownMemory struct {
			Enabled         bool     `json:"enabled"`
			Path            string   `json:"path"`
			SQLiteBacked    bool     `json:"sqlite_backed"`
			MarkdownBacked  bool     `json:"markdown_backed"`
			NetworkRequired bool     `json:"network_required"`
			OllamaRequired  bool     `json:"ollama_required"`
			MCPTools        []string `json:"mcp_tools"`
		} `json:"local_markdown_memory"`
		ToolRegistration struct {
			Registered []string `json:"registered"`
		} `json:"tool_registration"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v\nstdout=%s", err, stdout)
	}
	if !got.LocalMarkdownMemory.Enabled ||
		!got.LocalMarkdownMemory.SQLiteBacked ||
		!got.LocalMarkdownMemory.MarkdownBacked ||
		got.LocalMarkdownMemory.NetworkRequired ||
		got.LocalMarkdownMemory.OllamaRequired {
		t.Fatalf("local markdown memory = %+v, want enabled local-only markdown status", got.LocalMarkdownMemory)
	}
	if got.LocalMarkdownMemory.Path != filepath.Join(config.GormesHome(), "memory", "GONCHO_MEMORY.md") {
		t.Fatalf("local markdown path = %q", got.LocalMarkdownMemory.Path)
	}
	for _, want := range []string{"store_memory", "retrieve_memory", "update_memory", "summarize_memories", "forget_memory"} {
		if !slices.Contains(got.LocalMarkdownMemory.MCPTools, want) {
			t.Fatalf("local markdown mcp_tools = %#v, missing %s", got.LocalMarkdownMemory.MCPTools, want)
		}
		if !slices.Contains(got.ToolRegistration.Registered, want) {
			t.Fatalf("registered tools = %#v, missing %s", got.ToolRegistration.Registered, want)
		}
	}
}

func TestGonchoDoctorCommand_JSONReportsMemoryV1Contract(t *testing.T) {
	seedGonchoDoctorZeroStateDB(t)

	stdout, stderr, err := runGonchoDoctorCommand(t, "goncho", "doctor", "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		MemoryContract struct {
			ContractVersion                string          `json:"contract_version"`
			MarkdownFormatVersion          string          `json:"markdown_format_version"`
			MCPToolContractVersion         string          `json:"mcp_tool_contract_version"`
			PrivateAgentMemoryDefault      bool            `json:"private_agent_memory_default"`
			SelfImprovementPerAgentDefault bool            `json:"self_improvement_per_agent_default"`
			ForeignConfigRuntimeReads      string          `json:"foreign_config_runtime_reads"`
			FastRecallPath                 []string        `json:"fast_recall_path"`
			OptionalQualityLayers          []string        `json:"optional_quality_layers"`
			Tables                         map[string]bool `json:"tables"`
		} `json:"memory_contract"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v\nstdout=%s", err, stdout)
	}
	if got.MemoryContract.ContractVersion != "1" ||
		got.MemoryContract.MarkdownFormatVersion != "1" ||
		got.MemoryContract.MCPToolContractVersion != "1" {
		t.Fatalf("memory contract versions = %+v", got.MemoryContract)
	}
	if !got.MemoryContract.PrivateAgentMemoryDefault || !got.MemoryContract.SelfImprovementPerAgentDefault {
		t.Fatalf("memory contract isolation defaults = %+v", got.MemoryContract)
	}
	if got.MemoryContract.ForeignConfigRuntimeReads != "denied" {
		t.Fatalf("ForeignConfigRuntimeReads = %q, want denied", got.MemoryContract.ForeignConfigRuntimeReads)
	}
	for _, table := range []string{"goncho_memory_items", "goncho_memory_items_fts", "goncho_memory_eval_artifacts"} {
		if !got.MemoryContract.Tables[table] {
			t.Fatalf("memory_contract.tables[%s] = false, want true: %+v", table, got.MemoryContract.Tables)
		}
	}
}
