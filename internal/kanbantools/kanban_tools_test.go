package kanbantools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestKanbanToolsHiddenWithoutContextAndVisibleWithWorkerContext(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "kanban.db")
	if got := NewTools(Config{DBPath: dbPath}); len(got) != 0 {
		t.Fatalf("NewTools without worker context returned %d tools, want 0", len(got))
	}

	workerTools := NewTools(Config{DBPath: dbPath, TaskID: "t_worker", Profile: "coder"})
	assertKanbanToolNames(t, workerTools)

	profileToolsetTools := NewTools(Config{DBPath: dbPath, Enabled: true, Profile: "techlead"})
	assertKanbanToolNames(t, profileToolsetTools)
}

func TestKanbanToolsWorkerLifecycleStoresStructuredHandoff(t *testing.T) {
	ctx := context.Background()
	store, task := newClaimedKanbanTask(t, "worker-profile")
	defer store.Close()

	toolset := NewTools(Config{DBPath: store.DBPath(), TaskID: task.ID, Profile: "worker-profile"})

	show := executeKanbanTool(t, toolset, "kanban_show", `{}`)
	if taskPayload(show, "task")["id"] != task.ID {
		t.Fatalf("kanban_show task id = %v, want %s; payload=%v", taskPayload(show, "task")["id"], task.ID, show)
	}
	if _, ok := show["worker_context"].(string); !ok {
		t.Fatalf("kanban_show missing worker_context string: %v", show)
	}

	heartbeat := executeKanbanTool(t, toolset, "kanban_heartbeat", `{"note":"still compiling"}`)
	if heartbeat["ok"] != true {
		t.Fatalf("kanban_heartbeat = %v, want ok", heartbeat)
	}

	comment := executeKanbanTool(t, toolset, "kanban_comment", `{"body":"handoff note for this task"}`)
	if comment["ok"] != true || comment["comment_id"] == nil {
		t.Fatalf("kanban_comment = %v, want ok with comment_id", comment)
	}

	child := executeKanbanTool(t, toolset, "kanban_create", `{"title":"verify output","assignee":"qa","parents":["`+task.ID+`"],"body":"check the handoff"}`)
	childID, _ := child["task_id"].(string)
	if child["ok"] != true || childID == "" || child["status"] != "todo" {
		t.Fatalf("kanban_create = %v, want todo child task", child)
	}

	complete := executeKanbanTool(t, toolset, "kanban_complete", `{"summary":"implemented worker tools","metadata":{"changed_files":2,"tests":["unit"]}}`)
	if complete["ok"] != true {
		t.Fatalf("kanban_complete = %v, want ok", complete)
	}

	parent, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask(parent): %v", err)
	}
	if parent.Status != kanban.StatusDone {
		t.Fatalf("parent status = %q, want done", parent.Status)
	}
	readyChild, err := store.GetTask(ctx, childID)
	if err != nil {
		t.Fatalf("GetTask(child): %v", err)
	}
	if readyChild.Status != kanban.StatusReady {
		t.Fatalf("child status = %q, want ready after parent completion", readyChild.Status)
	}

	runs, err := store.ListRuns(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	latest := runs[len(runs)-1]
	if latest.Outcome != kanban.RunOutcomeCompleted || latest.Summary != "implemented worker tools" {
		t.Fatalf("latest run = %+v, want completed summary handoff", latest)
	}
	if !strings.Contains(string(latest.Metadata), `"changed_files":2`) {
		t.Fatalf("latest run metadata = %s, want changed_files", latest.Metadata)
	}

	comments, err := store.ListComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 1 || comments[0].Author != "worker-profile" {
		t.Fatalf("comments = %+v, want one worker-profile comment", comments)
	}
	events, err := store.ListEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if !hasKanbanEvent(events, "heartbeat") || !hasKanbanEvent(events, "completed") {
		t.Fatalf("events = %+v, want heartbeat and completed", events)
	}

	contextBlock, err := store.BuildWorkerContext(ctx, childID)
	if err != nil {
		t.Fatalf("BuildWorkerContext(child): %v", err)
	}
	for _, want := range []string{"implemented worker tools", "changed_files", "handoff note for this task"} {
		if !strings.Contains(contextBlock, want) {
			t.Fatalf("worker context missing %q:\n%s", want, contextBlock)
		}
	}
}

func TestKanbanToolsDenyForeignTaskMutationForWorkers(t *testing.T) {
	ctx := context.Background()
	store, own := newClaimedKanbanTask(t, "worker-profile")
	defer store.Close()
	foreign, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "foreign task", Assignee: "peer"})
	if err != nil {
		t.Fatalf("CreateTask(foreign): %v", err)
	}
	if _, claimed, err := store.ClaimTask(ctx, foreign.ID, kanban.ClaimTaskInput{Worker: "other", TTL: time.Minute}); err != nil || !claimed {
		t.Fatalf("ClaimTask(foreign) claimed=%v err=%v", claimed, err)
	}

	toolset := NewTools(Config{DBPath: store.DBPath(), TaskID: own.ID, Profile: "worker-profile"})
	for _, tc := range []struct {
		name string
		args string
	}{
		{"kanban_complete", `{"task_id":"` + foreign.ID + `","summary":"hijack"}`},
		{"kanban_block", `{"task_id":"` + foreign.ID + `","reason":"hijack"}`},
		{"kanban_heartbeat", `{"task_id":"` + foreign.ID + `"}`},
		{"kanban_comment", `{"task_id":"` + foreign.ID + `","body":"hijack"}`},
	} {
		out := executeKanbanTool(t, toolset, tc.name, tc.args)
		if out["ok"] == true || out["evidence"] != "kanban_task_ownership_denied" {
			t.Fatalf("%s foreign mutation = %v, want ownership denial", tc.name, out)
		}
	}

	got, err := store.GetTask(ctx, foreign.ID)
	if err != nil {
		t.Fatalf("GetTask(foreign): %v", err)
	}
	if got.Status != kanban.StatusRunning || got.Result != "" {
		t.Fatalf("foreign task mutated: %+v", got)
	}
	comments, err := store.ListComments(ctx, foreign.ID)
	if err != nil {
		t.Fatalf("ListComments(foreign): %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("foreign comments = %+v, want none", comments)
	}
}

func assertKanbanToolNames(t *testing.T, toolset []tools.Tool) {
	t.Helper()
	want := map[string]bool{
		"kanban_show":      true,
		"kanban_complete":  true,
		"kanban_block":     true,
		"kanban_heartbeat": true,
		"kanban_comment":   true,
		"kanban_create":    true,
		"kanban_link":      true,
	}
	if len(toolset) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(toolset), len(want))
	}
	for _, tool := range toolset {
		if !want[tool.Name()] {
			t.Fatalf("unexpected tool name %q", tool.Name())
		}
		delete(want, tool.Name())
	}
	if len(want) != 0 {
		t.Fatalf("missing tool names: %v", want)
	}
}

func newClaimedKanbanTask(t *testing.T, profile string) (*kanban.Store, kanban.Task) {
	t.Helper()
	ctx := context.Background()
	store, err := kanban.Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	task, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "worker task", Assignee: profile})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	claimed, ok, err := store.ClaimTask(ctx, task.ID, kanban.ClaimTaskInput{Worker: "kanban-dispatcher", TTL: time.Minute})
	if err != nil || !ok {
		t.Fatalf("ClaimTask: ok=%v err=%v", ok, err)
	}
	return store, claimed
}

func executeKanbanTool(t *testing.T, toolset []tools.Tool, name, args string) map[string]any {
	t.Helper()
	var selected tools.Tool
	for _, candidate := range toolset {
		if candidate.Name() == name {
			selected = candidate
			break
		}
	}
	if selected == nil {
		t.Fatalf("tool %s not found", name)
	}
	raw, err := selected.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s Execute error: %v\nraw=%s", name, err, raw)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s output unmarshal %s: %v", name, raw, err)
	}
	return out
}

func taskPayload(out map[string]any, key string) map[string]any {
	task, _ := out[key].(map[string]any)
	return task
}

func hasKanbanEvent(events []kanban.Event, kind string) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}
