package progress

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyHealthUpdates_RoundTripPreservesCheckedInProgressJSON loads the
// real checked-in progress.json, runs an empty update set through
// ApplyHealthUpdates, and asserts the file is byte-equal modulo trailing
// whitespace. Catches any field reordering or formatting drift.
func TestApplyHealthUpdates_RoundTripPreservesCheckedInProgressJSON(t *testing.T) {
	src := filepath.Join("..", "..", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	original, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("checked-in progress.json not found, skipping compat test: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(tmp, original, 0o644); err != nil {
		t.Fatalf("write tmp copy: %v", err)
	}

	if err := ApplyHealthUpdates(tmp, nil); err != nil {
		t.Fatalf("ApplyHealthUpdates with empty updates: %v", err)
	}

	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read tmp after round-trip: %v", err)
	}

	a := strings.TrimRight(string(original), "\n")
	b := strings.TrimRight(string(got), "\n")
	if a == b {
		return
	}
	// Surface a small unified diff at the first divergence.
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i] != b[i] {
			start := max(0, i-50)
			endA := min(len(a), i+50)
			endB := min(len(b), i+50)
			t.Fatalf("round-trip changed bytes at offset %d:\nORIG: %q\nGOT : %q", i,
				bytes.ReplaceAll([]byte(a[start:endA]), []byte("\n"), []byte("\\n")),
				bytes.ReplaceAll([]byte(b[start:endB]), []byte("\n"), []byte("\\n")))
		}
	}
	t.Fatalf("round-trip changed length: orig=%d got=%d", len(a), len(b))
}
