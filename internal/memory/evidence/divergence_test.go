package evidence

import "testing"

func TestVectorStoreDivergenceRowsClassifyExternalStoresAndReconciler(t *testing.T) {
	rows := VectorStoreDivergenceRows()
	if err := ValidateVectorStoreDivergence(rows); err != nil {
		t.Fatalf("ValidateVectorStoreDivergence: %v", err)
	}

	want := map[string]VectorDivergenceStatus{
		"lancedb_external_adapter":       VectorDivergenceExcluded,
		"turbopuffer_external_adapter":   VectorDivergenceExcluded,
		"vector_reconciler_sync_state":   VectorDivergenceExcluded,
		"sqlite_semantic_query_contract": VectorDivergenceOwned,
	}
	for _, row := range rows {
		status, ok := want[row.Name]
		if !ok {
			t.Fatalf("unexpected divergence row %q in %+v", row.Name, rows)
		}
		if row.Status != status {
			t.Fatalf("row %s status = %q, want %q", row.Name, row.Status, status)
		}
		delete(want, row.Name)
		if len(row.Guarantees) == 0 {
			t.Fatalf("row %s has no guarantees", row.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing divergence rows: %+v", want)
	}
}

func TestValidateVectorStoreDivergenceRejectsUnknownOrUnprovenRows(t *testing.T) {
	if err := ValidateVectorStoreDivergence([]VectorDivergenceRow{{
		Name:   "unknown",
		Status: "unknown",
		Upstream: []string{
			"../honcho/src/vector_store/lancedb.py",
		},
		Gormes: []string{
			"internal/memory/semantic_sql.go",
		},
		Rationale: "bad status",
	}}); err == nil {
		t.Fatal("ValidateVectorStoreDivergence accepted unknown status")
	}

	if err := ValidateVectorStoreDivergence([]VectorDivergenceRow{{
		Name:   "missing-evidence",
		Status: VectorDivergenceOwned,
	}}); err == nil {
		t.Fatal("ValidateVectorStoreDivergence accepted row without evidence")
	}
}
