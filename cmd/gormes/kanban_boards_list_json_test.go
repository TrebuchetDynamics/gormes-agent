package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestKanbanBoardsList_JSONEmitsEmptyArrayNotNull pins the regression
// observed during a fresh-install probe sweep:
// `gormes kanban boards list --json` on a fresh install emitted
// `"boards": null` instead of `"boards": []`. Fleet automation that
// iterates the array without nil-checks then crashes on `null`.
//
// Convention this enforces: read-only inventory `--json` surfaces
// always emit empty arrays, never null. Same shape as
// emitSessionListJSON returning `[]` for empty session inventories
// and collectSystemSnapshotForJSON normalizing nil events/presence.
func TestKanbanBoardsList_JSONEmitsEmptyArrayNotNull(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "kanban", "boards", "list", "--json")
	if err != nil {
		t.Fatalf("kanban boards list --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, `"boards": null`) || strings.Contains(stdout, `"boards":null`) {
		t.Fatalf(`"boards" must be `+"`[]`"+` on empty registry, not `+"`null`"+`; got:\n%s`, stdout)
	}

	var got struct {
		Boards []any `json:"boards"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Boards == nil {
		t.Fatalf(`"boards" must decode to []any{}, not nil; got %v\nstdout=%s`, got.Boards, stdout)
	}
	if len(got.Boards) != 0 {
		t.Errorf("got %d boards on fresh install, want 0", len(got.Boards))
	}
}
