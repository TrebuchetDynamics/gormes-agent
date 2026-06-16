package goncho_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho"
)

// A markdown item that omits created_at/updated_at passes ValidateGonchoMemoryV1Item
// (timestamps are not required), so Reload must default them rather than abort
// the whole all-or-nothing transaction when unixTime("") fails to parse.
func TestGonchoMarkdownStoreReloadDefaultsMissingTimestamps(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)

	doc := `---
goncho_memory_format: "1"
goncho_memory_contract: "1"
---

# Goncho Memory V1 No-Timestamp Fixture

<!-- goncho-memory
memory_id: mem_no_timestamps
revision: 1
agent_id: agent-a
workspace_id: workspace-private
peer_id: user-juan
scope: private
state: active
source_kind: manual
-->
An item authored without created_at or updated_at timestamps.
<!-- /goncho-memory -->
`
	markdownPath := filepath.Join(t.TempDir(), "GONCHO_MEMORY.md")
	if err := os.WriteFile(markdownPath, []byte(doc), 0o600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	markdown := goncho.NewGonchoMarkdownStore(store.DB(), goncho.GonchoMarkdownStoreConfig{
		Path:                  markdownPath,
		DefaultObserverPeerID: "agent-a",
	})
	reload, err := markdown.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload aborted on item with empty timestamps: %v", err)
	}
	if reload.Inserted != 1 || len(reload.Conflicts) != 0 {
		t.Fatalf("reload result = %+v, want 1 inserted, no conflicts", reload)
	}

	var createdAt, updatedAt int64
	if err := store.DB().QueryRow(`
		SELECT created_at, updated_at
		FROM goncho_memory_items
		WHERE memory_id = ?
	`, "mem_no_timestamps").Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("query item: %v", err)
	}
	if createdAt <= 0 || updatedAt <= 0 {
		t.Fatalf("created_at=%d updated_at=%d, want defaulted positive timestamps", createdAt, updatedAt)
	}
}
