package memory

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestGonchoMemoryV1ContractDeclaresStableCompatibilitySurface(t *testing.T) {
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
		if !stringSliceContains(contract.FastRecallPath, want) {
			t.Fatalf("FastRecallPath = %#v, missing %q", contract.FastRecallPath, want)
		}
	}
	for _, forbidden := range []string{"qmd", "embedding_generation", "remote_honcho", "hosted_api"} {
		if stringSliceContains(contract.FastRecallPath, forbidden) {
			t.Fatalf("FastRecallPath = %#v, must not require %q", contract.FastRecallPath, forbidden)
		}
	}
	for _, want := range []string{"qmd_deep_search", "embeddings", "dialectic", "dream_consolidation"} {
		if !stringSliceContains(contract.OptionalQualityLayers, want) {
			t.Fatalf("OptionalQualityLayers = %#v, missing %q", contract.OptionalQualityLayers, want)
		}
	}
}

func TestGonchoMemoryV1MarkdownRoundTripPreservesIDsScopesTombstonesAndChecksums(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "goncho_v1", "memory.md"))
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

func TestGonchoMemoryV1AgentIsolationAndSelfImprovementArtifacts(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "goncho_v1", "memory.md"))
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

	evalBody, err := os.ReadFile(filepath.Join("testdata", "goncho_v1", "eval_artifacts.jsonl"))
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

func TestGonchoMemoryV1MigrationBackfillsConclusionsAndCreatesIsolationTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	createSchema3iWithConclusion(t, path)

	store, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite migrate 3i->current: %v", err)
	}
	defer store.Close(context.Background())

	var version string
	if err := store.db.QueryRow(`SELECT v FROM schema_meta WHERE k = 'version'`).Scan(&version); err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %q, want %q", version, schemaVersion)
	}

	for _, table := range []string{"goncho_memory_items", "goncho_memory_items_fts", "goncho_memory_eval_artifacts"} {
		var found string
		if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil {
			t.Fatalf("table %s missing after migration: %v", table, err)
		}
	}

	var item GonchoMemoryV1Item
	var active int
	if err := store.db.QueryRow(`
		SELECT memory_id, agent_id, workspace_id, peer_id, session_key, content, revision, active, scope, provenance_json
		FROM goncho_memory_items
		WHERE memory_id = 'gmem_conclusion_42'
	`).Scan(&item.MemoryID, &item.AgentID, &item.WorkspaceID, &item.PeerID, &item.SessionID, &item.Content, &item.Revision, &active, &item.Scope, &item.ProvenanceJSON); err != nil {
		t.Fatalf("backfilled memory item: %v", err)
	}
	if item.AgentID != "agent-a" || item.WorkspaceID != "workspace-private" || item.PeerID != "user-juan" || item.SessionID != "telegram:1" {
		t.Fatalf("backfilled scope = %+v", item)
	}
	if item.Revision != 1 || active != 1 || item.Scope != "private" {
		t.Fatalf("backfilled revision/active/scope = revision %d active %d scope %q", item.Revision, active, item.Scope)
	}
	if !strings.Contains(item.ProvenanceJSON, `"source_table":"goncho_conclusions"`) || !strings.Contains(item.ProvenanceJSON, `"source_id":42`) {
		t.Fatalf("provenance_json = %s, want goncho_conclusions source evidence", item.ProvenanceJSON)
	}
}

func TestGonchoMemoryV1UnknownFutureVersionRefusesWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE schema_meta SET v = '999future' WHERE k = 'version'`); err != nil {
		t.Fatalf("force future version: %v", err)
	}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := OpenSqlite(path, 0, nil); err == nil {
		t.Fatal("OpenSqlite future version err = nil, want refusal")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open after failed migration: %v", err)
	}
	defer db.Close()
	var version string
	if err := db.QueryRow(`SELECT v FROM schema_meta WHERE k = 'version'`).Scan(&version); err != nil {
		t.Fatalf("schema version after failed migration: %v", err)
	}
	if version != "999future" {
		t.Fatalf("schema version after failed migration = %q, want untouched future version", version)
	}
}

func createSchema3iWithConclusion(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir db parent: %v", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if err := applyPragmas(db); err != nil {
		t.Fatalf("applyPragmas: %v", err)
	}
	if _, err := db.Exec(schemaV3a); err != nil {
		t.Fatalf("schemaV3a: %v", err)
	}
	for _, migration := range []struct {
		name string
		ddl  string
	}{
		{"3a->3b", migration3aTo3b},
		{"3b->3c", migration3bTo3c},
		{"3c->3d", migration3cTo3d},
		{"3d->3e", migration3dTo3e},
		{"3e->3f", migration3eTo3f},
		{"3f->3g", migration3fTo3g},
		{"3g->3h", migration3gTo3h},
		{"3h->3i", migration3hTo3i},
	} {
		if err := runMigrationTx(db, migration.ddl); err != nil {
			t.Fatalf("migration %s: %v", migration.name, err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO goncho_conclusions(
			id, workspace_id, observer_peer_id, peer_id, session_key, content, kind, status, source,
			idempotency_key, evidence_json, created_at, updated_at
		) VALUES(
			42, 'workspace-private', 'agent-a', 'user-juan', 'telegram:1',
			'Juan is focused on Goncho V1 memory safety.', 'manual', 'processed', 'manual',
			'fixture-42', '[{"turn_id":"turn-7"}]', 1700000000, 1700000100
		)
	`); err != nil {
		t.Fatalf("insert v3i conclusion: %v", err)
	}
}

func stringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
