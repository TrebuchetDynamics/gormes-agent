package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

func TestKanbanRunsCommand_JSONListsCompletedRunWithBuild(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	ctx := context.Background()

	task := seedKanbanRunTask(t, ctx, "json run task")
	store, err := openKanbanStore(ctx)
	if err != nil {
		t.Fatalf("openKanbanStore: %v", err)
	}
	defer store.Close()
	if err := store.CompleteTask(ctx, task.ID, kanban.CompleteTaskInput{
		Summary:  "handoff complete",
		Metadata: map[string]any{"commit": "abc123"},
	}); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "runs", task.ID, "--json")
	if err != nil {
		t.Fatalf("kanban runs --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		TaskID string           `json:"task_id"`
		Runs   []kanban.TaskRun `json:"runs"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("runs JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.TaskID != task.ID {
		t.Errorf("task_id = %q, want %q", got.TaskID, task.ID)
	}
	if len(got.Runs) != 1 {
		t.Fatalf("runs = %+v, want one run", got.Runs)
	}
	run := got.Runs[0]
	if run.Outcome != kanban.RunOutcomeCompleted {
		t.Errorf("run.outcome = %q, want %q", run.Outcome, kanban.RunOutcomeCompleted)
	}
	if run.Summary != "handoff complete" {
		t.Errorf("run.summary = %q, want handoff complete", run.Summary)
	}
	var metadata map[string]string
	if err := json.Unmarshal(run.Metadata, &metadata); err != nil {
		t.Fatalf("run.metadata decode: %v (%s)", err, string(run.Metadata))
	}
	if metadata["commit"] != "abc123" {
		t.Errorf("run.metadata.commit = %q, want abc123", metadata["commit"])
	}
	if run.StartedAt.IsZero() || run.EndedAt.IsZero() {
		t.Errorf("started_at/ended_at must be populated: %+v", run)
	}
}

func TestKanbanRunsCommand_JSONEmptyRunsArray(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	task := seedKanbanRunTask(t, context.Background(), "empty run task")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "runs", task.ID, "--json")
	if err != nil {
		t.Fatalf("kanban runs --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, `"runs": null`) || strings.Contains(stdout, `"runs":null`) {
		t.Fatalf(`"runs" must be [] on empty history, not null; stdout=%s`, stdout)
	}
	var got struct {
		Runs []kanban.TaskRun `json:"runs"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("runs JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Runs == nil {
		t.Fatalf("runs decoded to nil, want empty slice; stdout=%s", stdout)
	}
	if len(got.Runs) != 0 {
		t.Fatalf("runs length = %d, want 0", len(got.Runs))
	}
}

func TestKanbanRunsCommand_TextShowsOutcomeSummaryAndError(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	ctx := context.Background()
	task := seedKanbanRunTask(t, ctx, "text run task", "worker-a")
	store, err := openKanbanStore(ctx)
	if err != nil {
		t.Fatalf("openKanbanStore: %v", err)
	}
	defer store.Close()

	dispatcher := kanban.Dispatcher{
		Store: store,
		Spawner: kanban.SpawnFunc(func(context.Context, kanban.SpawnRequest) (kanban.SpawnResult, error) {
			return kanban.SpawnResult{}, errors.New("runner unavailable")
		}),
		Worker: "dispatcher",
	}
	if _, err := dispatcher.RunOnce(ctx, kanban.DispatchOptions{MaxSpawn: 1, FailureLimit: 3}); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if err := store.CompleteTask(ctx, task.ID, kanban.CompleteTaskInput{Summary: "final handoff"}); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "runs", task.ID)
	if err != nil {
		t.Fatalf("kanban runs: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"OUTCOME", "spawn_failed", "runner unavailable", "completed", "final handoff"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("kanban runs text missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"metadata"`) {
		t.Fatalf("human output must not dump raw metadata JSON:\n%s", stdout)
	}
}

func seedKanbanRunTask(t *testing.T, ctx context.Context, title string, assignee ...string) kanban.Task {
	t.Helper()
	store, err := openKanbanStore(ctx)
	if err != nil {
		t.Fatalf("openKanbanStore: %v", err)
	}
	defer store.Close()
	input := kanban.CreateTaskInput{Title: title}
	if len(assignee) > 0 {
		input.Assignee = assignee[0]
	}
	task, err := store.CreateTask(ctx, input)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	return task
}
