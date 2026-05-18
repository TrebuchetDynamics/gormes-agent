package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadInventoryDistinguishesGonchoDurableLegacyContextAndSessions(t *testing.T) {
	root := t.TempDir()
	mustWriteInventoryTestFile(t, filepath.Join(root, "memory", "USER.md"), "user memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "memories", "USER.md"), "legacy user\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "memories", "MEMORY.md"), "legacy memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "SOUL.md"), "profile soul\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "sessions", "index.yaml"), "sessions: {}\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "sessions", "20260518", "turn.md"), "transcript\n")

	store, err := OpenSqlite(filepath.Join(root, "memory.db"), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	now := time.Date(2026, 5, 18, 22, 30, 0, 0, time.UTC).Unix()
	_, err = store.DB().Exec(
		`INSERT INTO goncho_memory_items(
			memory_id, contract_version, agent_id, workspace_id, observer_peer_id,
			peer_id, session_key, source_kind, content, revision, active, scope,
			provenance_json, tags_json, importance, created_at, updated_at
		) VALUES
			('active', '1', 'gormes', 'default', 'gormes', 'user', 'sess', 'manual', 'active memory', 1, 1, 'private', '{}', '[]', 0.5, ?, ?),
			('tombstoned', '1', 'gormes', 'default', 'gormes', 'user', 'sess', 'manual', 'old memory', 1, 0, 'private', '{}', '[]', 0.5, ?, ?)`,
		now, now, now, now,
	)
	if err != nil {
		t.Fatalf("seed goncho memory items: %v", err)
	}

	got, err := ReadInventory(context.Background(), InventoryOptions{
		ProfileRoot: root,
		DB:          store.DB(),
		CWD:         root,
	})
	if err != nil {
		t.Fatalf("ReadInventory: %v", err)
	}

	if got.Goncho.ActiveItems != 1 {
		t.Fatalf("Goncho.ActiveItems = %d, want 1 active item only", got.Goncho.ActiveItems)
	}
	if got.DurableMarkdown.User.State != InventoryStatePresent {
		t.Fatalf("durable USER.md state = %q, want present", got.DurableMarkdown.User.State)
	}
	if got.DurableMarkdown.Memory.State != InventoryStateMissing {
		t.Fatalf("durable MEMORY.md state = %q, want missing", got.DurableMarkdown.Memory.State)
	}
	if got.LegacyHermes.User.State != InventoryStatePresent || got.LegacyHermes.Memory.State != InventoryStatePresent {
		t.Fatalf("legacy Hermes memory files = user:%q memory:%q, want both present", got.LegacyHermes.User.State, got.LegacyHermes.Memory.State)
	}
	if got.SelectedPromptMemoryDirRel != "memory" {
		t.Fatalf("SelectedPromptMemoryDirRel = %q, want memory", got.SelectedPromptMemoryDirRel)
	}
	if got.LegacyImportNeeded {
		t.Fatal("LegacyImportNeeded = true, want false when native durable markdown already exists")
	}
	if got.SessionTranscripts.Files != 2 {
		t.Fatalf("SessionTranscripts.Files = %d, want 2", got.SessionTranscripts.Files)
	}
	if !inventoryContextHas(got.ContextFiles, "SOUL.md", InventoryStatePresent) {
		t.Fatalf("ContextFiles = %+v, want present SOUL.md evidence", got.ContextFiles)
	}
}

func TestReadInventoryFlagsLegacyImportNeededWhenOnlyHermesMemoriesExist(t *testing.T) {
	root := t.TempDir()
	mustWriteInventoryTestFile(t, filepath.Join(root, "memories", "USER.md"), "legacy user\n")

	got, err := ReadInventory(context.Background(), InventoryOptions{
		ProfileRoot: root,
		CWD:         root,
	})
	if err != nil {
		t.Fatalf("ReadInventory: %v", err)
	}

	if !got.LegacyImportNeeded {
		t.Fatal("LegacyImportNeeded = false, want true when legacy Hermes memories exist without native Gormes memory")
	}
	if got.SelectedPromptMemoryDirRel != "memories" {
		t.Fatalf("SelectedPromptMemoryDirRel = %q, want memories", got.SelectedPromptMemoryDirRel)
	}
}

func inventoryContextHas(items []InventoryFile, rel string, state InventoryState) bool {
	for _, item := range items {
		if item.RelativePath == rel && item.State == state {
			return true
		}
	}
	return false
}

func mustWriteInventoryTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
