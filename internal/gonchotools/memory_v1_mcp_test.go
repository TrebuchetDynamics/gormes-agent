package gonchotools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho"
)

func TestMemoryV1MCPTranscriptUsesOnlyContractedTools(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "memory", "testdata", "goncho_v1", "tool_transcript.jsonl"))
	if err != nil {
		t.Fatalf("read tool transcript fixture: %v", err)
	}
	entries, err := goncho.DecodeMemoryV1ToolTranscript(body)
	if err != nil {
		t.Fatalf("DecodeMemoryV1ToolTranscript: %v", err)
	}
	contract := goncho.MemoryV1ToolContract()

	seen := map[string]bool{}
	for _, entry := range entries {
		spec, ok := contract.Tools[entry.Tool]
		if !ok {
			t.Fatalf("fixture references non-contract tool %q", entry.Tool)
		}
		seen[entry.Tool] = true
		if entry.Result["contract_version"] != contract.ContractVersion {
			t.Fatalf("%s contract_version = %#v, want %q", entry.Tool, entry.Result["contract_version"], contract.ContractVersion)
		}
		if spec.Mutating && !spec.RequiresProvenance {
			t.Fatalf("%s is mutating without provenance requirement: %+v", entry.Tool, spec)
		}
	}
	for name := range contract.Tools {
		if !seen[name] {
			t.Fatalf("fixture missing contracted tool %s", name)
		}
	}
}

func TestMemoryV1MCPContractKeepsPurgeOutOfNormalToolSet(t *testing.T) {
	contract := goncho.MemoryV1ToolContract()
	if _, ok := contract.Tools["purge_memory"]; ok {
		t.Fatalf("purge_memory present in normal V1 tool set: %+v", contract.Tools["purge_memory"])
	}
	if contract.PurgePolicy != "explicit_operator_only" {
		t.Fatalf("PurgePolicy = %q, want explicit_operator_only", contract.PurgePolicy)
	}
}
