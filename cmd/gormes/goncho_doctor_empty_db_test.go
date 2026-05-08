package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestGonchoDoctor_ZeroByteMemoryDB_EmitsStructuredReport pins the
// regression observed during a fresh-install probe sweep:
// `gormes goncho doctor --json --peer=test` against a 0-byte
// `memory.db` leaked the raw sqlite error
// `sqlite3: SQL logic error: no such table: schema_meta` to the
// operator. That looks like a corrupt-DB bug, not the
// "schema not applied yet" diagnostic it actually represents.
//
// Contract: ReadSchemaStatus errors that say "no such table" mean
// the DB exists but the goncho schema hasn't been applied — route
// these into the same `runtime_storage_error` exit-2 structured
// report the existing `corrupt_memory_database_is_runtime_storage_issue`
// test asserts. Operators see the structured diagnostic ladder; raw
// sqlite text never surfaces.
func TestGonchoDoctor_ZeroByteMemoryDB_EmitsStructuredReport(t *testing.T) {
	setupGonchoDoctorEnv(t)

	if err := os.MkdirAll(filepath.Dir(config.MemoryDBPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	// Touch a 0-byte memory.db (sqlite opens this fine, but no
	// goncho schema is present).
	if err := os.WriteFile(config.MemoryDBPath(), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runGonchoDoctorCommand(t, "goncho", "doctor", "--json", "--peer=test")
	// Exit 2 is correct: 0-byte DB is a runtime storage issue —
	// schema isn't applied. The structured report ladder gives the
	// operator something to act on; the raw sqlite error doesn't.
	if code := commandExitCode(err); code != 2 {
		t.Fatalf("exit code = %d, want 2\nstdout=%s\nstderr=%s\nerr=%v", code, stdout, stderr, err)
	}

	// Stderr must NOT carry the raw sqlite text. (Cobra renders
	// returned errors via stderr.)
	combined := stdout + stderr
	if strings.Contains(combined, "no such table") || strings.Contains(combined, "SQL logic error") {
		t.Fatalf("raw sqlite error must not leak; got:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	// Stdout must carry a structured report (--json contract).
	var got struct {
		Service  string `json:"service"`
		Status   string `json:"status"`
		ExitCode int    `json:"exit_code"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON for --json mode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Status != "runtime_storage_error" {
		t.Errorf("status = %q, want %q (schema not applied is a storage-state issue)", got.Status, "runtime_storage_error")
	}
	if got.ExitCode != 2 {
		t.Errorf("exit_code field = %d, want 2", got.ExitCode)
	}
}
