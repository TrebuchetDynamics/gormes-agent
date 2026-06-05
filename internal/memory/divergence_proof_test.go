package memory

import (
	"context"
	"slices"
	"testing"
)

func TestVectorStoreDivergenceProof_LocalSemanticCacheHasNoExternalSyncState(t *testing.T) {
	store := openMemoryStoreForDivergence(t)
	defer store.Close(context.Background())

	columns := sqliteColumns(t, store, "entity_embeddings")
	for _, forbidden := range []string{"sync_state", "sync_attempts", "last_sync_at"} {
		if slices.Contains(columns, forbidden) {
			t.Fatalf("entity_embeddings unexpectedly has Honcho external sync column %q in %v", forbidden, columns)
		}
	}

	cache := NewSemanticCache()
	ids := seedEmbeddedGraph(t, store, "local-model", 4, []string{"Local"})
	query := []float32{1, 0, 0, 0}
	got, err := semanticSeeds(context.Background(), store.db, cache, "local-model", query, 5, 0.8)
	if err != nil {
		t.Fatalf("semanticSeeds: %v", err)
	}
	if len(got) != 1 || got[0] != ids["Local"] {
		t.Fatalf("semanticSeeds = %v, want local SQLite entity %d", got, ids["Local"])
	}
}

func openMemoryStoreForDivergence(t *testing.T) *SqliteStore {
	t.Helper()
	store, err := OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	return store
}

func sqliteColumns(t *testing.T, store *SqliteStore, table string) []string {
	t.Helper()
	rows, err := store.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info(%s): %v", table, err)
	}
	return out
}
