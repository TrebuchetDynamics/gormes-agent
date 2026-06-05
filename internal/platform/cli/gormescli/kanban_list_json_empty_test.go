package gormescli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestKanbanList_JSONEmitsEmptyArrayNotNull pins the regression
// observed during a fresh-install probe sweep:
// `gormes kanban list --json` on an empty board emitted
// `"tasks": null` instead of `"tasks": []`. Fleet automation
// iterating the array (`for t in d['tasks']`, `data.tasks.length`)
// crashes on null/None.
//
// Convention enforced: read-only inventory `--json` surfaces always
// emit empty arrays, never null. Already enforced for sessions
// (emitSessionListJSON) and kanban boards (slice 32). This slice
// closes the same gap on the primary kanban list.
func TestKanbanList_JSONEmitsEmptyArrayNotNull(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "kanban", "list", "--json")
	if err != nil {
		t.Fatalf("kanban list --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, `"tasks": null`) || strings.Contains(stdout, `"tasks":null`) {
		t.Fatalf(`"tasks" must be `+"`[]`"+` on empty board, not `+"`null`"+`; got:\n%s`, stdout)
	}

	var got struct {
		Tasks []any `json:"tasks"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Tasks == nil {
		t.Fatalf(`"tasks" must decode to []any{}, not nil; got %v\nstdout=%s`, got.Tasks, stdout)
	}
}
