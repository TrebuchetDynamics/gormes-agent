package kanbantools

import (
	"context"
	"database/sql"
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
	assertKanbanLifecycleToolNames(t, workerTools)

	profileToolsetTools := NewTools(Config{DBPath: dbPath, Enabled: true, Profile: "techlead"})
	assertKanbanOrchestratorToolNames(t, profileToolsetTools)

	workerWithToolset := NewTools(Config{DBPath: dbPath, TaskID: "t_worker", Enabled: true, Profile: "coder"})
	assertKanbanLifecycleToolNames(t, workerWithToolset)
	for _, banned := range []string{"kanban_list", "kanban_unblock"} {
		if hasKanbanTool(workerWithToolset, banned) {
			t.Fatalf("%s registered for scoped worker with kanban toolset", banned)
		}
	}
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

func TestKanbanToolsCommentIgnoresCallerSuppliedAuthor(t *testing.T) {
	ctx := context.Background()
	store, task := newClaimedKanbanTask(t, "worker-profile")
	defer store.Close()

	toolset := NewTools(Config{DBPath: store.DBPath(), TaskID: task.ID, Profile: "worker-profile"})
	comment := executeKanbanTool(t, toolset, "kanban_comment", `{"body":"handoff note","author":"hermes-system"}`)
	if comment["ok"] != true {
		t.Fatalf("kanban_comment = %v, want ok", comment)
	}

	comments, err := store.ListComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("comments = %+v, want one", comments)
	}
	if comments[0].Author != "worker-profile" {
		t.Fatalf("comment author = %q, want worker-profile", comments[0].Author)
	}
}

func TestKanbanToolsCommentSchemaOmitsAuthorOverride(t *testing.T) {
	store, task := newClaimedKanbanTask(t, "worker-profile")
	defer store.Close()

	toolset := NewTools(Config{DBPath: store.DBPath(), TaskID: task.ID, Profile: "worker-profile"})
	commentTool := findKanbanTool(t, toolset, "kanban_comment")
	var schema struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(commentTool.Schema(), &schema); err != nil {
		t.Fatalf("comment schema JSON: %v", err)
	}
	if _, ok := schema.Properties["author"]; ok {
		t.Fatalf("kanban_comment schema exposes author override: %s", commentTool.Schema())
	}
}

func TestKanbanToolsWorkerCanCommentOnForeignTask(t *testing.T) {
	ctx := context.Background()
	store, own := newClaimedKanbanTask(t, "worker-profile")
	defer store.Close()
	foreign, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "foreign task", Assignee: "peer"})
	if err != nil {
		t.Fatalf("CreateTask(foreign): %v", err)
	}

	toolset := NewTools(Config{DBPath: store.DBPath(), TaskID: own.ID, Profile: "worker-profile"})
	comment := executeKanbanTool(t, toolset, "kanban_comment", `{"task_id":"`+foreign.ID+`","body":"handoff: see prior findings","author":"hermes-system"}`)
	if comment["ok"] != true {
		t.Fatalf("cross-task kanban_comment = %v, want ok", comment)
	}

	comments, err := store.ListComments(ctx, foreign.ID)
	if err != nil {
		t.Fatalf("ListComments(foreign): %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("foreign comments = %+v, want one", comments)
	}
	if comments[0].Author != "worker-profile" {
		t.Fatalf("foreign comment author = %q, want worker-profile", comments[0].Author)
	}
	if !strings.HasPrefix(comments[0].Body, "handoff:") {
		t.Fatalf("foreign comment body = %q, want handoff prefix", comments[0].Body)
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

func TestKanbanToolsOrchestratorListFiltersAndBooleanArgs(t *testing.T) {
	ctx := context.Background()
	store, err := kanban.Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	alpha, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "alpha", Assignee: "factory", Priority: 5})
	if err != nil {
		t.Fatalf("CreateTask(alpha): %v", err)
	}
	beta, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "beta", Assignee: "factory"})
	if err != nil {
		t.Fatalf("CreateTask(beta): %v", err)
	}
	if _, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "review", Assignee: "reviewer"}); err != nil {
		t.Fatalf("CreateTask(review): %v", err)
	}
	archived, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "archived", Assignee: "factory"})
	if err != nil {
		t.Fatalf("CreateTask(archived): %v", err)
	}
	markTaskStatus(t, store.DBPath(), archived.ID, kanban.StatusArchived)

	toolset := NewTools(Config{DBPath: store.DBPath(), Enabled: true, Profile: "techlead"})
	list := executeKanbanTool(t, toolset, "kanban_list", `{"assignee":"factory","status":"ready","include_archived":"false","limit":1}`)
	if list["ok"] != true {
		t.Fatalf("kanban_list = %v, want ok", list)
	}
	if list["count"] != float64(1) || list["limit"] != float64(1) || list["truncated"] != true || list["next_limit"] != float64(2) {
		t.Fatalf("kanban_list paging = %v, want count=1 limit=1 truncated next_limit=2", list)
	}
	tasks := listTasksPayload(t, list)
	if taskID(tasks[0]) != alpha.ID {
		t.Fatalf("first filtered task id = %v, want %s", taskID(tasks[0]), alpha.ID)
	}
	if taskID(tasks[0]) == archived.ID {
		t.Fatalf("archived task leaked with include_archived=false: %v", tasks[0])
	}

	withArchived := executeKanbanTool(t, toolset, "kanban_list", `{"assignee":"factory","include_archived":"true","limit":10}`)
	if withArchived["ok"] != true {
		t.Fatalf("kanban_list include archived = %v, want ok", withArchived)
	}
	ids := listTaskIDs(t, withArchived)
	for _, want := range []string{alpha.ID, beta.ID, archived.ID} {
		if !containsString(ids, want) {
			t.Fatalf("include_archived ids = %v, missing %s", ids, want)
		}
	}

	badBool := executeKanbanTool(t, toolset, "kanban_list", `{"include_archived":"sometimes"}`)
	if badBool["ok"] == true || !strings.Contains(stringValue(badBool["error"]), "include_archived") {
		t.Fatalf("bad include_archived = %v, want bounded bool error", badBool)
	}

	badLimit := executeKanbanTool(t, toolset, "kanban_list", `{"limit":201}`)
	if badLimit["ok"] == true || !strings.Contains(stringValue(badLimit["error"]), "limit") {
		t.Fatalf("bad limit = %v, want bounded limit error", badLimit)
	}
}

func TestKanbanToolsOrchestratorUnblockRecomputesReadiness(t *testing.T) {
	ctx := context.Background()
	store, err := kanban.Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	parent, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "parent"})
	if err != nil {
		t.Fatalf("CreateTask(parent): %v", err)
	}
	child, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "child", ParentIDs: []string{parent.ID}})
	if err != nil {
		t.Fatalf("CreateTask(child): %v", err)
	}
	if err := store.BlockTask(ctx, child.ID, kanban.BlockTaskInput{Reason: "waiting"}); err != nil {
		t.Fatalf("BlockTask(child): %v", err)
	}

	toolset := NewTools(Config{DBPath: store.DBPath(), Enabled: true, Profile: "techlead"})
	unblock := executeKanbanTool(t, toolset, "kanban_unblock", `{"task_id":"`+child.ID+`"}`)
	if unblock["ok"] != true || unblock["status"] != string(kanban.StatusTodo) {
		t.Fatalf("kanban_unblock = %v, want ok todo", unblock)
	}

	nonBlocked := executeKanbanTool(t, toolset, "kanban_unblock", `{"task_id":"`+parent.ID+`"}`)
	if nonBlocked["ok"] == true || !strings.Contains(stringValue(nonBlocked["error"]), "not blocked") {
		t.Fatalf("kanban_unblock non-blocked = %v, want bounded not-blocked error", nonBlocked)
	}
}

func TestKanbanToolsCreateParsesTriageStringBooleans(t *testing.T) {
	ctx := context.Background()
	store, err := kanban.Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	toolset := NewTools(Config{DBPath: store.DBPath(), Enabled: true, Profile: "techlead"})
	ready := executeKanbanTool(t, toolset, "kanban_create", `{"title":"ready task","assignee":"coder","triage":"false"}`)
	if ready["ok"] != true {
		t.Fatalf("kanban_create triage false = %v, want ok", ready)
	}
	readyTask, err := store.GetTask(ctx, stringValue(ready["task_id"]))
	if err != nil {
		t.Fatalf("GetTask(ready): %v", err)
	}
	if readyTask.Status != kanban.StatusReady {
		t.Fatalf("triage false status = %q, want ready", readyTask.Status)
	}

	triage := executeKanbanTool(t, toolset, "kanban_create", `{"title":"rough idea","assignee":"coder","triage":"true"}`)
	if triage["ok"] != true {
		t.Fatalf("kanban_create triage true = %v, want ok", triage)
	}
	triageTask, err := store.GetTask(ctx, stringValue(triage["task_id"]))
	if err != nil {
		t.Fatalf("GetTask(triage): %v", err)
	}
	if triageTask.Status != kanban.StatusTriage {
		t.Fatalf("triage true status = %q, want triage", triageTask.Status)
	}

	bad := executeKanbanTool(t, toolset, "kanban_create", `{"title":"bad","assignee":"coder","triage":"sometimes"}`)
	if bad["ok"] == true || !strings.Contains(stringValue(bad["error"]), "triage") {
		t.Fatalf("kanban_create bad triage = %v, want bounded bool error", bad)
	}
}

func findKanbanTool(t *testing.T, toolset []tools.Tool, name string) tools.Tool {
	t.Helper()
	for _, candidate := range toolset {
		if candidate.Name() == name {
			return candidate
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil
}

func assertKanbanLifecycleToolNames(t *testing.T, toolset []tools.Tool) {
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

func assertKanbanOrchestratorToolNames(t *testing.T, toolset []tools.Tool) {
	t.Helper()
	want := map[string]bool{
		"kanban_show":      true,
		"kanban_list":      true,
		"kanban_complete":  true,
		"kanban_block":     true,
		"kanban_heartbeat": true,
		"kanban_comment":   true,
		"kanban_create":    true,
		"kanban_link":      true,
		"kanban_unblock":   true,
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

func hasKanbanTool(toolset []tools.Tool, name string) bool {
	for _, tool := range toolset {
		if tool.Name() == name {
			return true
		}
	}
	return false
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

func listTasksPayload(t *testing.T, out map[string]any) []any {
	t.Helper()
	tasks, ok := out["tasks"].([]any)
	if !ok {
		t.Fatalf("tasks payload = %T %v, want []any", out["tasks"], out["tasks"])
	}
	return tasks
}

func listTaskIDs(t *testing.T, out map[string]any) []string {
	t.Helper()
	tasks := listTasksPayload(t, out)
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskMap, ok := task.(map[string]any)
		if !ok {
			t.Fatalf("task payload = %T %v, want map", task, task)
		}
		ids = append(ids, taskID(taskMap))
	}
	return ids
}

func taskID(task any) string {
	taskMap, _ := task.(map[string]any)
	id, _ := taskMap["id"].(string)
	return id
}

func markTaskStatus(t *testing.T, dbPath, id string, status kanban.Status) {
	t.Helper()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite fixture: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE tasks SET status = ? WHERE id = ?`, string(status), id); err != nil {
		t.Fatalf("mark task %s status %s: %v", id, status, err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasKanbanEvent(events []kanban.Event, kind string) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}
