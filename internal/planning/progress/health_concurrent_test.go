package progress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestApplyHealthUpdates_ConcurrentDisjointKeysAllSucceed runs N goroutines
// each writing a disjoint row's health block. ApplyHealthUpdates has no
// internal locking — last writer wins per the documented single-writer
// semantics — so this test pins two weaker guarantees:
//
//  1. No write returns an error (atomic temp+rename means each write either
//     fully lands or fully aborts; nothing partial gets committed).
//  2. The file is still parseable afterward and at least one health block
//     survived. Lost updates from concurrent writers are expected; corrupt
//     output is not.
func TestApplyHealthUpdates_ConcurrentDisjointKeysAllSucceed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")

	// Build a progress doc with N rows.
	const N = 8
	body := `{"version":"1","phases":{"1":{"name":"P","subphases":{"1.A":{"name":"S","items":[`
	for i := 0; i < N; i++ {
		if i > 0 {
			body += ","
		}
		body += fmt.Sprintf(`{"name":"row-%d","status":"planned"}`, i)
	}
	body += `]}}}}}`
	writeProgressJSON(t, path, body)

	var wg sync.WaitGroup
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := ApplyHealthUpdates(path, []HealthUpdate{{
				PhaseID:    "1",
				SubphaseID: "1.A",
				ItemName:   fmt.Sprintf("row-%d", idx),
				Mutate: func(h *RowHealth) {
					h.AttemptCount = idx + 1
				},
			}})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent write returned error: %v", err)
		}
	}

	// File must still be parseable; no partial state.
	_, err := Load(path)
	if err != nil {
		t.Fatalf("Load after concurrent writes: %v", err)
	}
	// At least one update must have survived (last writer wins; others may be lost
	// because the helper has no locking — that's expected single-writer semantics).
	body2, _ := os.ReadFile(path)
	if !strings.Contains(string(body2), `"attempt_count":`) {
		t.Fatalf("no health block survived concurrent writes:\n%s", body2)
	}
}
