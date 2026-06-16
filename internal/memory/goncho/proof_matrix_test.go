package goncho_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho/testutil"
)

func TestGonchoProofMatrix_MemoryV1GovernanceTombstonesAndRecallContracts(t *testing.T) {
	contract := goncho.GonchoMemoryV1Contract()
	if contract.ContractVersion != "1" || contract.MarkdownFormatVersion != "1" || contract.MCPToolContractVersion != "1" {
		t.Fatalf("contract versions = %+v, want v1 across storage/markdown/MCP", contract)
	}
	if !contract.PrivateAgentMemoryDefault || contract.ForeignConfigRuntimeReads != "denied" {
		t.Fatalf("governance defaults = %+v, want private memory and denied foreign config reads", contract)
	}
	for _, want := range []string{"sqlite", "fts5", "graph"} {
		if !testutil.StringSliceContains(contract.FastRecallPath, want) {
			t.Fatalf("FastRecallPath = %+v, missing %s", contract.FastRecallPath, want)
		}
	}

	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())
	status, err := goncho.ReadGonchoMemoryV1Status(context.Background(), store.DB())
	if err != nil {
		t.Fatalf("ReadGonchoMemoryV1Status: %v", err)
	}
	for _, table := range []string{"goncho_memory_items", "goncho_memory_items_fts", "goncho_memory_eval_artifacts"} {
		if !status.Tables[table] {
			t.Fatalf("status tables = %+v, missing %s", status.Tables, table)
		}
	}

	body, err := os.ReadFile(filepath.Join("..", "testdata", "goncho_v1", "memory.md"))
	if err != nil {
		t.Fatalf("read memory fixture: %v", err)
	}
	doc, err := goncho.ParseGonchoMemoryV1Markdown(body)
	if err != nil {
		t.Fatalf("ParseGonchoMemoryV1Markdown: %v", err)
	}
	items := map[string]goncho.GonchoMemoryV1Item{}
	for _, item := range doc.Items {
		if err := goncho.ValidateGonchoMemoryV1Item(item); err != nil {
			t.Fatalf("goncho.ValidateGonchoMemoryV1Item(%s): %v", item.MemoryID, err)
		}
		items[item.MemoryID] = item
	}

	ownerReq := goncho.GonchoMemoryV1RecallRequest{AgentID: "agent-a", WorkspaceID: "workspace-private"}
	if allowed, reason := goncho.CanRecallGonchoMemoryV1(ownerReq, items["mem_agent_a_project"]); !allowed || reason != "owner_agent" {
		t.Fatalf("owner recall allowed=%t reason=%q", allowed, reason)
	}
	foreignReq := goncho.GonchoMemoryV1RecallRequest{AgentID: "agent-b", WorkspaceID: "workspace-private"}
	if allowed, reason := goncho.CanRecallGonchoMemoryV1(foreignReq, items["mem_agent_a_project"]); allowed || reason != "private_agent_boundary" {
		t.Fatalf("foreign recall allowed=%t reason=%q", allowed, reason)
	}
	sharedReq := goncho.GonchoMemoryV1RecallRequest{AgentID: "agent-b", WorkspaceID: "workspace-shared", AllowShared: true}
	if allowed, reason := goncho.CanRecallGonchoMemoryV1(sharedReq, items["mem_shared_standard"]); !allowed || reason != "shared_workspace" {
		t.Fatalf("shared recall allowed=%t reason=%q", allowed, reason)
	}
	if allowed, reason := goncho.CanRecallGonchoMemoryV1(ownerReq, items["mem_agent_a_old_goal"]); allowed || reason != "tombstoned" {
		t.Fatalf("tombstoned recall allowed=%t reason=%q", allowed, reason)
	}
	if items["mem_agent_a_old_goal"].TombstoneReason == "" || items["mem_agent_a_old_goal"].TombstonedAt == "" {
		t.Fatalf("tombstone item = %+v, want reason and timestamp", items["mem_agent_a_old_goal"])
	}

	evalBody, err := os.ReadFile(filepath.Join("..", "testdata", "goncho_v1", "eval_artifacts.jsonl"))
	if err != nil {
		t.Fatalf("read eval fixture: %v", err)
	}
	artifacts, err := goncho.DecodeGonchoMemoryV1EvalArtifacts(evalBody)
	if err != nil {
		t.Fatalf("DecodeGonchoMemoryV1EvalArtifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("artifacts = %d, want 2", len(artifacts))
	}
	for _, artifact := range artifacts {
		if artifact.AgentID == "" || artifact.WorkspaceID == "" || artifact.SourceMemoryID == "" {
			t.Fatalf("artifact lacks provenance scope: %+v", artifact)
		}
		if artifact.Shared && artifact.Status != "proposed" {
			t.Fatalf("shared artifact = %+v, want proposed before shared mutation", artifact)
		}
	}
}
