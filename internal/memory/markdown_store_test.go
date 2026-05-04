package memory

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGonchoMarkdownStoreReloadExportAndConflictHandling(t *testing.T) {
	ctx := context.Background()
	store, err := OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)

	fixture, err := os.ReadFile(filepath.Join("testdata", "goncho_v1", "memory.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	markdownPath := filepath.Join(t.TempDir(), "GONCHO_MEMORY.md")
	if err := os.WriteFile(markdownPath, fixture, 0o600); err != nil {
		t.Fatalf("write markdown fixture: %v", err)
	}

	markdown := NewGonchoMarkdownStore(store.DB(), GonchoMarkdownStoreConfig{
		Path:                  markdownPath,
		DefaultObserverPeerID: "agent-a",
	})
	reload, err := markdown.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if reload.NetworkRequired || reload.OllamaRequired {
		t.Fatalf("reload required optional remote/vector layers: %+v", reload)
	}
	if reload.Inserted != 3 || reload.Tombstoned != 1 || len(reload.Conflicts) != 0 {
		t.Fatalf("reload result = %+v, want 3 inserted, 1 tombstoned, no conflicts", reload)
	}
	assertGonchoMemoryItemContent(t, store.DB(), "mem_agent_a_project", "fast local recall", 2, true)
	assertGonchoMemoryItemContent(t, store.DB(), "mem_agent_a_old_goal", "QMD required", 3, false)

	exportPath := filepath.Join(t.TempDir(), "exported.md")
	markdown.Config.Path = exportPath
	exported, err := markdown.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if exported.Exported != 3 || exported.NetworkRequired || exported.OllamaRequired {
		t.Fatalf("export result = %+v, want 3 local exported items", exported)
	}
	roundTripBody, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	roundTrip, err := ParseGonchoMemoryV1Markdown(roundTripBody)
	if err != nil {
		t.Fatalf("parse exported markdown: %v\n%s", err, roundTripBody)
	}
	if len(roundTrip.Items) != 3 {
		t.Fatalf("exported items = %d, want 3", len(roundTrip.Items))
	}

	editedBody := strings.Replace(string(fixture),
		"Juan is focused on Goncho V1 memory safety and wants fast local recall.",
		"Juan is focused on Goncho V1 memory safety and wants edited markdown recall.",
		1,
	)
	editPath := filepath.Join(t.TempDir(), "edited.md")
	if err := os.WriteFile(editPath, []byte(editedBody), 0o600); err != nil {
		t.Fatalf("write edited markdown: %v", err)
	}
	markdown.Config.Path = editPath
	edited, err := markdown.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload edited file: %v", err)
	}
	if len(edited.Conflicts) != 0 {
		t.Fatalf("edit conflicts = %+v, want same-revision human edit to reload", edited.Conflicts)
	}
	assertGonchoMemoryItemContent(t, store.DB(), "mem_agent_a_project", "edited markdown recall", 3, true)

	conflictBody := strings.Replace(string(fixture),
		"Juan is focused on Goncho V1 memory safety and wants fast local recall.",
		"Juan is focused on Goncho V1 memory safety and wants stale markdown recall.",
		1,
	)
	conflictPath := filepath.Join(t.TempDir(), "conflict.md")
	if err := os.WriteFile(conflictPath, []byte(conflictBody), 0o600); err != nil {
		t.Fatalf("write conflict markdown: %v", err)
	}
	markdown.Config.Path = conflictPath
	conflicted, err := markdown.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload conflict file: %v", err)
	}
	if len(conflicted.Conflicts) != 1 {
		t.Fatalf("conflicts = %+v, want one stale revision conflict", conflicted.Conflicts)
	}
	if conflicted.Conflicts[0].MemoryID != "mem_agent_a_project" || conflicted.Conflicts[0].Reason != "stale_revision" {
		t.Fatalf("conflict = %+v, want mem_agent_a_project stale_revision", conflicted.Conflicts[0])
	}
	assertGonchoMemoryItemContent(t, store.DB(), "mem_agent_a_project", "edited markdown recall", 3, true)
}

func assertGonchoMemoryItemContent(t *testing.T, db *sql.DB, memoryID, wantFragment string, wantRevision int, wantActive bool) {
	t.Helper()

	var content string
	var revision int
	var active int
	if err := db.QueryRow(`
		SELECT content, revision, active
		FROM goncho_memory_items
		WHERE memory_id = ?
	`, memoryID).Scan(&content, &revision, &active); err != nil {
		t.Fatalf("query %s: %v", memoryID, err)
	}
	if !strings.Contains(content, wantFragment) || revision != wantRevision || (active == 1) != wantActive {
		t.Fatalf("%s content=%q revision=%d active=%d, want fragment %q revision %d active %t", memoryID, content, revision, active, wantFragment, wantRevision, wantActive)
	}
}
