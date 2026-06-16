package goncho

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho/testutil"
)

func TestMemoryV1ContractDeclaresStableCompatibilitySurface(t *testing.T) {
	contract := GonchoMemoryV1Contract()

	if contract.ContractVersion != "1" {
		t.Fatalf("ContractVersion = %q, want 1", contract.ContractVersion)
	}
	if contract.MarkdownFormatVersion != "1" {
		t.Fatalf("MarkdownFormatVersion = %q, want 1", contract.MarkdownFormatVersion)
	}
	if contract.MCPToolContractVersion != "1" {
		t.Fatalf("MCPToolContractVersion = %q, want 1", contract.MCPToolContractVersion)
	}
	if !contract.PrivateAgentMemoryDefault || !contract.SelfImprovementPerAgentDefault {
		t.Fatalf("agent isolation defaults = private %t self_improvement %t, want both true", contract.PrivateAgentMemoryDefault, contract.SelfImprovementPerAgentDefault)
	}
	for _, want := range []string{"sqlite", "fts5", "graph"} {
		if !testutil.StringSliceContains(contract.FastRecallPath, want) {
			t.Fatalf("FastRecallPath = %#v, missing %q", contract.FastRecallPath, want)
		}
	}
	for _, forbidden := range []string{"qmd", "embedding_generation", "remote_honcho", "hosted_api"} {
		if testutil.StringSliceContains(contract.FastRecallPath, forbidden) {
			t.Fatalf("FastRecallPath = %#v, must not require %q", contract.FastRecallPath, forbidden)
		}
	}
	for _, want := range []string{"qmd_deep_search", "embeddings", "dialectic", "dream_consolidation"} {
		if !testutil.StringSliceContains(contract.OptionalQualityLayers, want) {
			t.Fatalf("OptionalQualityLayers = %#v, missing %q", contract.OptionalQualityLayers, want)
		}
	}
}

func TestMemoryV1MarkdownRoundTripPreservesIDsScopesTombstonesAndChecksums(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "testdata", "goncho_v1", "memory.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	doc, err := ParseGonchoMemoryV1Markdown(body)
	if err != nil {
		t.Fatalf("ParseGonchoMemoryV1Markdown: %v", err)
	}
	if doc.FormatVersion != "1" || doc.ContractVersion != "1" {
		t.Fatalf("versions = format %q contract %q, want 1/1", doc.FormatVersion, doc.ContractVersion)
	}
	if len(doc.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(doc.Items))
	}

	items := map[string]GonchoMemoryV1Item{}
	for _, item := range doc.Items {
		if err := ValidateGonchoMemoryV1Item(item); err != nil {
			t.Fatalf("ValidateGonchoMemoryV1Item(%s): %v", item.MemoryID, err)
		}
		items[item.MemoryID] = item
	}

	if got := items["mem_agent_a_project"].AgentID; got != "agent-a" {
		t.Fatalf("agent-a item AgentID = %q", got)
	}
	if got := items["mem_agent_a_project"].Scope; got != "private" {
		t.Fatalf("agent-a item Scope = %q, want private", got)
	}
	if got := items["mem_shared_standard"].Scope; got != "shared" {
		t.Fatalf("shared item Scope = %q, want shared", got)
	}
	if items["mem_agent_a_old_goal"].State != "tombstoned" || items["mem_agent_a_old_goal"].TombstonedAt == "" {
		t.Fatalf("tombstoned item = %+v, want tombstoned with timestamp", items["mem_agent_a_old_goal"])
	}

	rendered, err := RenderGonchoMemoryV1Markdown(doc)
	if err != nil {
		t.Fatalf("RenderGonchoMemoryV1Markdown: %v", err)
	}
	roundTrip, err := ParseGonchoMemoryV1Markdown([]byte(rendered))
	if err != nil {
		t.Fatalf("round-trip parse: %v\n%s", err, rendered)
	}
	if len(roundTrip.Items) != len(doc.Items) {
		t.Fatalf("round-trip items = %d, want %d", len(roundTrip.Items), len(doc.Items))
	}
	for _, item := range roundTrip.Items {
		if err := ValidateGonchoMemoryV1Item(item); err != nil {
			t.Fatalf("round-trip ValidateGonchoMemoryV1Item(%s): %v", item.MemoryID, err)
		}
	}
}

func TestMemoryV1AgentIsolationAndSelfImprovementArtifacts(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "testdata", "goncho_v1", "memory.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := ParseGonchoMemoryV1Markdown(body)
	if err != nil {
		t.Fatalf("ParseGonchoMemoryV1Markdown: %v", err)
	}
	items := map[string]GonchoMemoryV1Item{}
	for _, item := range doc.Items {
		items[item.MemoryID] = item
	}

	privateReq := GonchoMemoryV1RecallRequest{AgentID: "agent-b", WorkspaceID: "workspace-private"}
	if allowed, reason := CanRecallGonchoMemoryV1(privateReq, items["mem_agent_a_project"]); allowed || reason != "private_agent_boundary" {
		t.Fatalf("agent-b private recall allowed=%t reason=%q, want false private_agent_boundary", allowed, reason)
	}

	ownerReq := GonchoMemoryV1RecallRequest{AgentID: "agent-a", WorkspaceID: "workspace-private"}
	if allowed, reason := CanRecallGonchoMemoryV1(ownerReq, items["mem_agent_a_project"]); !allowed || reason != "owner_agent" {
		t.Fatalf("owner private recall allowed=%t reason=%q, want true owner_agent", allowed, reason)
	}

	sharedReq := GonchoMemoryV1RecallRequest{AgentID: "agent-b", WorkspaceID: "workspace-shared", AllowShared: true}
	if allowed, reason := CanRecallGonchoMemoryV1(sharedReq, items["mem_shared_standard"]); !allowed || reason != "shared_workspace" {
		t.Fatalf("shared recall allowed=%t reason=%q, want true shared_workspace", allowed, reason)
	}

	if allowed, reason := CanRecallGonchoMemoryV1(ownerReq, items["mem_agent_a_old_goal"]); allowed || reason != "tombstoned" {
		t.Fatalf("tombstone recall allowed=%t reason=%q, want false tombstoned", allowed, reason)
	}

	evalBody, err := os.ReadFile(filepath.Join("..", "testdata", "goncho_v1", "eval_artifacts.jsonl"))
	if err != nil {
		t.Fatalf("read eval fixture: %v", err)
	}
	artifacts, err := DecodeGonchoMemoryV1EvalArtifacts(evalBody)
	if err != nil {
		t.Fatalf("DecodeGonchoMemoryV1EvalArtifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("artifacts = %d, want 2", len(artifacts))
	}
	for _, artifact := range artifacts {
		if artifact.AgentID == "" || artifact.WorkspaceID == "" {
			t.Fatalf("artifact missing per-agent scope: %+v", artifact)
		}
		if artifact.Shared && artifact.Status != "proposed" {
			t.Fatalf("shared artifact = %+v, want reviewed/proposed path before shared mutation", artifact)
		}
	}
}
