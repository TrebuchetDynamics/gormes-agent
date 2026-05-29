package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
)

func TestKanbanStatsCommand_JSONIncludesBuildAndStats(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	ready := runKanbanJSONTask(t, "create", "ready stats task", "--assignee", "alice", "--json")
	parent := runKanbanJSONTask(t, "create", "stats parent", "--json")
	child := runKanbanJSONTask(t, "create", "todo stats child", "--parent", parent.ID, "--assignee", "alice", "--json")
	blocked := runKanbanJSONTask(t, "create", "blocked stats task", "--assignee", "bob", "--json")
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "block", blocked.ID, "waiting", "--json"); err != nil {
		t.Fatalf("kanban block: %v\nstderr=%s", err, stderr)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "stats", "--json")
	if err != nil {
		t.Fatalf("kanban stats --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		ByStatus              map[string]int            `json:"by_status"`
		ByAssignee            map[string]map[string]int `json:"by_assignee"`
		OldestReadyAgeSeconds *int64                    `json:"oldest_ready_age_seconds"`
		Now                   int64                     `json:"now"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stats JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.ByStatus[string(kanban.StatusReady)] != 2 {
		t.Fatalf("ready count = %d, want 2 (ready=%s parent=%s)", got.ByStatus[string(kanban.StatusReady)], ready.ID, parent.ID)
	}
	if got.ByStatus[string(kanban.StatusTodo)] != 1 {
		t.Fatalf("todo count = %d, want 1 (child=%s)", got.ByStatus[string(kanban.StatusTodo)], child.ID)
	}
	if got.ByStatus[string(kanban.StatusBlocked)] != 1 {
		t.Fatalf("blocked count = %d, want 1", got.ByStatus[string(kanban.StatusBlocked)])
	}
	if got.ByAssignee["alice"][string(kanban.StatusReady)] != 1 || got.ByAssignee["alice"][string(kanban.StatusTodo)] != 1 {
		t.Fatalf("alice counts = %+v, want ready=1 todo=1", got.ByAssignee["alice"])
	}
	if got.ByAssignee["bob"][string(kanban.StatusBlocked)] != 1 {
		t.Fatalf("bob counts = %+v, want blocked=1", got.ByAssignee["bob"])
	}
	if got.OldestReadyAgeSeconds == nil || *got.OldestReadyAgeSeconds < 0 {
		t.Fatalf("oldest_ready_age_seconds = %v, want non-negative value", got.OldestReadyAgeSeconds)
	}
	if got.Now <= 0 {
		t.Fatalf("now = %d, want unix timestamp", got.Now)
	}
}

func TestKanbanStatsCommand_TextMatchesHermesOperatorShape(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	_ = runKanbanJSONTask(t, "create", "ready stats task", "--assignee", "alice", "--json")
	blocked := runKanbanJSONTask(t, "create", "blocked stats task", "--assignee", "bob", "--json")
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "block", blocked.ID, "waiting"); err != nil {
		t.Fatalf("kanban block: %v\nstderr=%s", err, stderr)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "stats")
	if err != nil {
		t.Fatalf("kanban stats: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"By status:",
		"ready",
		"blocked",
		"By assignee:",
		"alice",
		"ready=1",
		"bob",
		"blocked=1",
		"Oldest ready task age:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("kanban stats text missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "archived") {
		t.Fatalf("human stats output must not print archived counts:\n%s", stdout)
	}
}

func TestKanbanStatsCommand_JSONEmptyMapsNotNull(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "stats", "--json")
	if err != nil {
		t.Fatalf("kanban stats --json empty: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, `"by_status": null`) || strings.Contains(stdout, `"by_assignee": null`) {
		t.Fatalf("empty stats maps must encode as objects, not null:\n%s", stdout)
	}
	var got struct {
		ByStatus              map[string]int            `json:"by_status"`
		ByAssignee            map[string]map[string]int `json:"by_assignee"`
		OldestReadyAgeSeconds *int64                    `json:"oldest_ready_age_seconds"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty stats JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.ByStatus == nil || got.ByAssignee == nil {
		t.Fatalf("decoded stats maps = %#v %#v, want non-nil empty maps", got.ByStatus, got.ByAssignee)
	}
	if got.OldestReadyAgeSeconds != nil {
		t.Fatalf("oldest_ready_age_seconds = %v, want null", *got.OldestReadyAgeSeconds)
	}
}
