package plannertriggers

import (
	"path/filepath"
	"sync"
	"testing"
)

// TestAppendTriggerEvent_ConcurrentWritersAllSucceed exercises the
// append-only contract under contention: N goroutines append in parallel
// and we verify that every event landed and that the auto-generated IDs
// are unique. This is the property the planner cursor depends on (each
// event must have a stable, distinct ID).
func TestAppendTriggerEvent_ConcurrentWritersAllSucceed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "triggers.jsonl")
	const N = 8
	var wg sync.WaitGroup
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := AppendTriggerEvent(path, TriggerEvent{
				Kind:    "quarantine_added",
				PhaseID: "p",
			}); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent append error: %v", err)
	}
	all, err := ReadTriggersSinceCursor(path, TriggerCursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != N {
		t.Fatalf("expected %d events, got %d", N, len(all))
	}
	seen := map[string]bool{}
	for _, ev := range all {
		if ev.ID == "" {
			t.Fatalf("event missing ID: %+v", ev)
		}
		if seen[ev.ID] {
			t.Fatalf("duplicate ID: %s", ev.ID)
		}
		seen[ev.ID] = true
	}
}
