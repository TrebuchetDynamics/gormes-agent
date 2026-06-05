package gormescli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestKanbanBoardsList_JSONEmitsDefaultNotNull pins two fresh-install
// contracts: the board array is never null, and the implicit default
// board is visible even before a named board exists.
//
// Convention this enforces: read-only inventory `--json` surfaces
// always emit arrays, never null. Same shape as emitSessionListJSON
// returning `[]` for empty session inventories and
// collectSystemSnapshotForJSON normalizing nil events/presence.
func TestKanbanBoardsList_JSONEmitsDefaultNotNull(t *testing.T) {
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
		Current string `json:"current"`
		Boards  []struct {
			Name    string `json:"name"`
			Current bool   `json:"current"`
			Total   int    `json:"total"`
		} `json:"boards"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Boards == nil {
		t.Fatalf(`"boards" must decode to []any{}, not nil; got %v\nstdout=%s`, got.Boards, stdout)
	}
	if got.Current != "default" {
		t.Errorf("current = %q, want default", got.Current)
	}
	if len(got.Boards) != 1 {
		t.Fatalf("got %d boards on fresh install, want implicit default board", len(got.Boards))
	}
	if got.Boards[0].Name != "default" || !got.Boards[0].Current || got.Boards[0].Total != 0 {
		t.Fatalf("fresh board = %+v, want current default with zero tasks", got.Boards[0])
	}
}
