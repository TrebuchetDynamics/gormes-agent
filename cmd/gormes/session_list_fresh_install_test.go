package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionList_MissingMemoryDB_EmitsEmptyState pins the regression
// observed on every fresh install: `gormes session list` errored with
// `memory database not found at /home/xel/.gormes/memory.db` and exit
// 1, instead of the friendly "No sessions found." UX it ships when
// the directory is empty.
//
// Slice 19 already fixed the case where memory.db EXISTS but the
// `turns` table doesn't; this fence covers the prior step where the
// DB file itself doesn't exist (the goncho memory store creates it
// lazily on first turn write). For an inventory command the absence
// of state is not an error — it's the empty state.
func TestSessionList_MissingMemoryDB_EmitsEmptyState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "list")
	if err != nil {
		t.Fatalf("session list on fresh install must succeed; got err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "No sessions found.") {
		t.Fatalf("stdout must read \"No sessions found.\"; got %q", stdout)
	}
}

// TestSessionList_MissingMemoryDB_JSONEmitsEmptyArray keeps the JSON
// surface symmetric: fleet automation parsing `gormes session list
// --json` from a freshly-imaged host should see `{"sessions": []}`,
// not a non-zero exit + free-form error string.
func TestSessionList_MissingMemoryDB_JSONEmitsEmptyArray(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "list", "--json")
	if err != nil {
		t.Fatalf("session list --json on fresh install must succeed; got err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Sessions []any `json:"sessions"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Sessions == nil {
		t.Fatalf("sessions must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("got %d sessions, want 0", len(got.Sessions))
	}
}

func TestSessionList_CorruptMemoryDBSelfHealsToEmptyArray(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes-home")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	if err := os.MkdirAll(gormesHome, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(gormesHome, "memory.db")
	if err := os.WriteFile(dbPath, []byte("not sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "list", "--json")
	if err != nil {
		t.Fatalf("session list --json must self-heal corrupt memory.db; got err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Sessions []any `json:"sessions"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Sessions == nil || len(got.Sessions) != 0 {
		t.Fatalf("sessions must be empty array after self-heal; stdout=%s", stdout)
	}
	combined := stdout + stderr
	if strings.Contains(combined, "file is not a database") || strings.Contains(combined, "SQL logic error") {
		t.Fatalf("raw sqlite corruption leaked to operator: %q", combined)
	}
	backups, err := filepath.Glob(dbPath + ".corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("corrupt memory.db must be preserved as one quarantine backup, got %v", backups)
	}
}

// TestOpenSessionDirectoryDB_ExportPathStillErrors keeps the
// regression fence: making `session list` soft on a missing DB must
// not weaken the export/delete/continue paths, where a missing DB
// means "the operator asked to read/mutate a session that can't
// exist" — that's a real error.
func TestOpenSessionDirectoryDB_ExportPathStillErrors(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))

	// Sanity: the home is empty, no memory.db.
	if _, err := os.Stat(filepath.Join(root, "gormes-home", "memory.db")); !os.IsNotExist(err) {
		t.Fatalf("test setup leaked a memory.db; stat err=%v", err)
	}

	_, stderr, err := runSessionsCommand(t, nil, "session", "export", "any-id")
	if err == nil {
		t.Fatalf("session export on fresh install must error (no DB to read from); stderr=%s", stderr)
	}
	if !strings.Contains(err.Error(), "memory database not found") {
		t.Fatalf("export error must say `memory database not found`; got %q", err)
	}
}
