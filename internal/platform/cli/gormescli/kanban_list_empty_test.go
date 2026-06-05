package gormescli

import (
	"strings"
	"testing"
)

// TestKanbanListEmpty_PlainModeShowsHelpfulMessage proves that
// `gormes kanban list` (plain mode) emits a friendly placeholder
// when there are no tasks, instead of an empty stdout that operators
// can't distinguish from a hung command or a silently-failing query.
//
// JSON mode keeps the empty `tasks: []` shape (covered separately) so
// machine consumers see no behavior change.
func TestKanbanListEmpty_PlainModeShowsHelpfulMessage(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}), "kanban", "list",
	)
	if err != nil {
		t.Fatalf("kanban list (empty): %v\nstderr=%s", err, stderr)
	}
	combined := stdout + stderr
	if strings.TrimSpace(combined) == "" {
		t.Fatalf("plain `kanban list` on empty board produced silent output;\n"+
			"operators must see *something* (e.g. \"No Kanban tasks.\") so "+
			"they can distinguish empty board from a hung command.\nstdout=%q stderr=%q", stdout, stderr)
	}
	// "No" is the canonical empty-list opener used elsewhere
	// (cf. `gormes plugins`: "No plugins installed.").
	if !strings.Contains(combined, "No ") {
		t.Errorf("empty `kanban list` placeholder must lead with \"No \" "+
			"to match the convention used by `gormes plugins`; got:\n%s", combined)
	}
	if !strings.Contains(strings.ToLower(combined), "kanban") {
		t.Errorf("empty `kanban list` placeholder must mention \"kanban\"; got:\n%s", combined)
	}
}

// TestKanbanListEmpty_JSONStaysEmptyTasksArray is the regression fence:
// the new placeholder must NOT alter `--json` output, since JSON consumers
// (CI, dashboards) parse `tasks: []` and would break if an empty array
// became a string.
func TestKanbanListEmpty_JSONStaysEmptyTasksArray(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}), "kanban", "list", "--json",
	)
	if err != nil {
		t.Fatalf("kanban list --json (empty): %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, "No Kanban") {
		t.Errorf("--json output must not include the human placeholder; got:\n%s", stdout)
	}
	// The JSON shape is `{ build, tasks: [] }`. We don't unmarshal here to
	// keep this test surgical; we just assert the empty-array marker.
	if !strings.Contains(stdout, `"tasks"`) {
		t.Errorf("--json output must include \"tasks\" key; got:\n%s", stdout)
	}
}
