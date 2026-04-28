package memory

import (
	"errors"
	"fmt"
)

type VectorDivergenceStatus string

const (
	VectorDivergenceOwned    VectorDivergenceStatus = "owned"
	VectorDivergenceExcluded VectorDivergenceStatus = "excluded"
)

// VectorDivergenceRow records the intentional split between hosted Honcho's
// external vector-store/reconciler model and Gormes' local SQLite semantic
// memory. It is runtime evidence, not a queue of work.
type VectorDivergenceRow struct {
	Name       string                 `json:"name"`
	Status     VectorDivergenceStatus `json:"status"`
	Upstream   []string               `json:"upstream"`
	Gormes     []string               `json:"gormes"`
	Rationale  string                 `json:"rationale"`
	Guarantees []string               `json:"guarantees,omitempty"`
}

func VectorStoreDivergenceRows() []VectorDivergenceRow {
	rows := []VectorDivergenceRow{
		{
			Name:   "lancedb_external_adapter",
			Status: VectorDivergenceExcluded,
			Upstream: []string{
				"../honcho/src/vector_store/lancedb.py:LanceDBVectorStore",
				"../honcho/src/vector_store/__init__.py:VectorStore",
			},
			Gormes: []string{
				"internal/memory/schema.go:migration3cTo3d",
				"internal/memory/semantic_sql.go:semanticSeeds",
			},
			Rationale: "Gormes stores normalized semantic vectors in local SQLite entity_embeddings rows instead of creating LanceDB tables/namespaces.",
			Guarantees: []string{
				"no LanceDB path or table namespace is required for local recall",
				"SQLite foreign-key cascade owns cleanup with entity deletion",
			},
		},
		{
			Name:   "turbopuffer_external_adapter",
			Status: VectorDivergenceExcluded,
			Upstream: []string{
				"../honcho/src/vector_store/turbopuffer.py:TurbopufferVectorStore",
				"../honcho/src/config.py:VECTOR_STORE_TURBOPUFFER_API_KEY",
			},
			Gormes: []string{
				"internal/memory/schema.go:migration3cTo3d",
				"internal/memory/semantic_sql.go:semanticSeeds",
			},
			Rationale: "Gormes does not require a hosted vector API key or remote namespace for Goncho recall.",
			Guarantees: []string{
				"missing external vector credentials cannot disable local semantic recall",
				"provider/network failures are outside the local memory query path",
			},
		},
		{
			Name:   "vector_reconciler_sync_state",
			Status: VectorDivergenceExcluded,
			Upstream: []string{
				"../honcho/src/reconciler/sync_vectors.py:ReconciliationMetrics",
				"../honcho/src/reconciler/sync_vectors.py:_get_documents_needing_sync",
				"../honcho/src/reconciler/sync_vectors.py:_get_message_embeddings_needing_sync",
			},
			Gormes: []string{
				"internal/memory/semantic_sql.go:semanticCache",
				"internal/memory/semantic_sql.go:semanticSeeds",
			},
			Rationale: "Gormes has no external-vector pending/synced/failed reconciliation state; cache generation invalidation is the local consistency contract.",
			Guarantees: []string{
				"entity_embeddings has no sync_state, sync_attempts, or last_sync_at columns",
				"semantic cache rebuilds on graph-version changes",
			},
		},
		{
			Name:   "sqlite_semantic_query_contract",
			Status: VectorDivergenceOwned,
			Upstream: []string{
				"../honcho/src/vector_store/lancedb.py:query",
				"../honcho/src/vector_store/turbopuffer.py:query",
			},
			Gormes: []string{
				"internal/memory/semantic_sql.go:semanticSeeds",
				"internal/memory/cosine.go",
			},
			Rationale: "Gormes owns local Top-K cosine search over normalized vectors, model filtering, and corrupt-row tolerance.",
			Guarantees: []string{
				"queries filter by embedding model",
				"dimension mismatches and corrupt BLOB rows are skipped rather than fatal",
				"results are ordered by similarity and capped to the requested Top-K",
			},
		},
	}
	out := make([]VectorDivergenceRow, len(rows))
	copy(out, rows)
	return out
}

func ValidateVectorStoreDivergence(rows []VectorDivergenceRow) error {
	if len(rows) == 0 {
		return errors.New("memory: vector divergence rows are required")
	}
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.Name == "" {
			return errors.New("memory: vector divergence row missing name")
		}
		if seen[row.Name] {
			return fmt.Errorf("memory: duplicate vector divergence row %q", row.Name)
		}
		seen[row.Name] = true
		switch row.Status {
		case VectorDivergenceOwned, VectorDivergenceExcluded:
		default:
			return fmt.Errorf("memory: vector divergence row %q has unknown status %q", row.Name, row.Status)
		}
		if len(row.Upstream) == 0 || len(row.Gormes) == 0 || row.Rationale == "" {
			return fmt.Errorf("memory: vector divergence row %q missing evidence", row.Name)
		}
	}
	return nil
}
