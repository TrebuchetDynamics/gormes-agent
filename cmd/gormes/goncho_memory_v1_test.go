package main

import (
	"encoding/json"
	"strings"
	"testing"
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
