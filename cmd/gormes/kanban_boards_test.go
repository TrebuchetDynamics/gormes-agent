package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
)

type kanbanBoardListReportForTest struct {
	Build struct {
		Version string `json:"version"`
	} `json:"build"`
	Current string                      `json:"current"`
	Boards  []kanbanBoardSummaryForTest `json:"boards"`
}

type kanbanBoardSummaryForTest struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	Current bool           `json:"current"`
	Counts  map[string]int `json:"counts"`
	Total   int            `json:"total"`
}

func TestKanbanBoardsListJSONIncludesDefaultCurrentCountsAndAliases(t *testing.T) {
	root := t.TempDir()
	hermesHome := filepath.Join(t.TempDir(), "hermes")
	t.Setenv("GORMES_HOME", root)
	t.Setenv("HERMES_HOME", hermesHome)

	defaultTask := runKanbanJSONTask(t, "create", "default board task", "--json")
	if defaultTask.Status != kanban.StatusReady {
		t.Fatalf("default task status = %q, want ready", defaultTask.Status)
	}

	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "create", "alpha"); err != nil {
		t.Fatalf("kanban boards create alpha: %v\nstderr=%s", err, stderr)
	}
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "switch", "alpha"); err != nil {
		t.Fatalf("kanban boards switch alpha: %v\nstderr=%s", err, stderr)
	}

	alphaReady := runKanbanJSONTask(t, "create", "alpha ready", "--json")
	alphaBlocked := runKanbanJSONTask(t, "create", "alpha blocked", "--json")
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "block", alphaBlocked.ID, "waiting", "--json"); err != nil {
		t.Fatalf("kanban block alpha task: %v\nstderr=%s", err, stderr)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "ls", "--json")
	if err != nil {
		t.Fatalf("kanban boards ls --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got kanbanBoardListReportForTest
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("boards list JSON decode: %v\nstdout=%s", err, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Current != "alpha" {
		t.Fatalf("current = %q, want alpha", got.Current)
	}

	defaultBoard := requireKanbanBoardSummary(t, got.Boards, "default")
	alphaBoard := requireKanbanBoardSummary(t, got.Boards, "alpha")
	if defaultBoard.Current {
		t.Fatalf("default current = true, want false")
	}
	if !alphaBoard.Current {
		t.Fatalf("alpha current = false, want true")
	}
	if defaultBoard.Total != 1 || defaultBoard.Counts[string(kanban.StatusReady)] != 1 {
		t.Fatalf("default counts = total %d %+v, want one ready task", defaultBoard.Total, defaultBoard.Counts)
	}
	if alphaBoard.Total != 2 ||
		alphaBoard.Counts[string(kanban.StatusReady)] != 1 ||
		alphaBoard.Counts[string(kanban.StatusBlocked)] != 1 {
		t.Fatalf("alpha counts = total %d %+v, want one ready and one blocked task (%s, %s)", alphaBoard.Total, alphaBoard.Counts, alphaReady.ID, alphaBlocked.ID)
	}
	if _, err := os.Stat(filepath.Join(hermesHome, "kanban.db")); !os.IsNotExist(err) {
		t.Fatalf("Hermes kanban state was touched, stat err = %v", err)
	}
}

func TestKanbanBoardsCurrentAndShowReportCountsWithoutCreatingDefaultDB(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)

	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "create", "alpha"); err != nil {
		t.Fatalf("kanban boards create alpha: %v\nstderr=%s", err, stderr)
	}
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "switch", "alpha"); err != nil {
		t.Fatalf("kanban boards switch alpha: %v\nstderr=%s", err, stderr)
	}
	alphaTask := runKanbanJSONTask(t, "create", "alpha read model", "--json")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "current", "--json")
	if err != nil {
		t.Fatalf("kanban boards current --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Current string                    `json:"current"`
		Board   kanbanBoardSummaryForTest `json:"board"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("boards current JSON decode: %v\nstdout=%s", err, stdout)
	}
	if got.Current != "alpha" || got.Board.Name != "alpha" || !got.Board.Current {
		t.Fatalf("current report = %+v, want current alpha board", got)
	}
	if got.Board.Total != 1 || got.Board.Counts[string(kanban.StatusReady)] != 1 {
		t.Fatalf("current board counts = total %d %+v, want task %s", got.Board.Total, got.Board.Counts, alphaTask.ID)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "show", "default")
	if err != nil {
		t.Fatalf("kanban boards show default: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Board: default", "Tasks: 0 total"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("show default output missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "kanban.db")); !os.IsNotExist(err) {
		t.Fatalf("show default created the default DB, stat err = %v", err)
	}
}

func TestKanbanBoardsShowMissingBoardFailsWithoutMutation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "show", "missing")
	if err == nil {
		t.Fatalf("kanban boards show missing error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, `board "missing" does not exist`) {
		t.Fatalf("missing-board output missing not-found evidence:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	if _, err := os.Stat(filepath.Join(root, "kanban", "boards", "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing board directory was created, stat err = %v", err)
	}
}

func requireKanbanBoardSummary(t *testing.T, boards []kanbanBoardSummaryForTest, name string) kanbanBoardSummaryForTest {
	t.Helper()
	for _, board := range boards {
		if board.Name == name {
			return board
		}
	}
	t.Fatalf("board %q not found in %+v", name, boards)
	return kanbanBoardSummaryForTest{}
}
