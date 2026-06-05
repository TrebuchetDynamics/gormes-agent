package provenance

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestReadInventoryDistinguishesGonchoDurableLegacyContextAndSessions(t *testing.T) {
	root := t.TempDir()
	mustWriteInventoryTestFile(t, filepath.Join(root, "memory", "USER.md"), "user memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "memories", "USER.md"), "legacy user\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "memories", "MEMORY.md"), "legacy memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "SOUL.md"), "profile soul\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "sessions", "index.yaml"), "sessions: {}\n")
	mustWriteInventoryTestFile(t, filepath.Join(root, "sessions", "20260518", "turn.md"), "transcript\n")

	db, err := sql.Open("sqlite3", filepath.Join(root, "memory.db"))
	if err != nil {
		t.Fatalf("open sqlite fixture: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE goncho_memory_items (memory_id TEXT PRIMARY KEY, active INTEGER, created_at INTEGER, updated_at INTEGER)`); err != nil {
		t.Fatalf("create goncho_memory_items fixture: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE turns (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create turns fixture: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE goncho_memory_eval_artifacts (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create goncho_memory_eval_artifacts fixture: %v", err)
	}

	now := time.Date(2026, 5, 18, 22, 30, 0, 0, time.UTC).Unix()
	_, err = db.Exec(
		`INSERT INTO goncho_memory_items(memory_id, active, created_at, updated_at) VALUES
			('active', 1, ?, ?),
			('tombstoned', 0, ?, ?)`,
		now, now, now, now,
	)
	if err != nil {
		t.Fatalf("seed goncho memory items: %v", err)
	}

	got, err := ReadInventory(context.Background(), InventoryOptions{
		ProfileRoot: root,
		DB:          db,
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
