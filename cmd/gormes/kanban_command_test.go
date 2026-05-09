package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

func TestKanbanCommandUsesGormesHomeNotHermesHome(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes")
	hermesHome := filepath.Join(root, "hermes")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", hermesHome)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "init")
	if err != nil {
		t.Fatalf("kanban init error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, filepath.Join(gormesHome, "kanban.db")) {
		t.Fatalf("kanban init stdout = %q, want GORMES_HOME kanban.db", stdout)
	}
	if _, err := os.Stat(filepath.Join(gormesHome, "kanban.db")); err != nil {
		t.Fatalf("Gormes kanban.db not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hermesHome, "kanban.db")); !os.IsNotExist(err) {
		t.Fatalf("Hermes kanban.db touched, stat err = %v", err)
	}
}

func TestKanbanCommandCreateWithParentPromotesAfterComplete(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	parent := runKanbanJSONTask(t, "create", "Design schema", "--assignee", "researcher", "--json")
	if parent.Status != kanban.StatusReady || parent.Assignee != "researcher" {
		t.Fatalf("parent = %+v, want ready researcher", parent)
	}

	child := runKanbanJSONTask(t, "create", "Implement API", "--assignee", "backend-dev", "--parent", parent.ID, "--json")
	if child.Status != kanban.StatusTodo {
		t.Fatalf("child status = %q, want %q", child.Status, kanban.StatusTodo)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "complete", parent.ID, "--result", "schema ready")
	if err != nil {
		t.Fatalf("kanban complete error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	got := runKanbanJSONTask(t, "show", child.ID, "--json")
	if got.Status != kanban.StatusReady {
		t.Fatalf("child after complete status = %q, want %q", got.Status, kanban.StatusReady)
	}
}

func TestKanbanSpecifyCommandUsesAuxiliarySpecifier(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	restore := stubKanbanTriageSpecifier(&recordingCommandTriageSpecifier{
		response: `{"title":"Specify release task","body":"**Goal** - make it shippable\n**Acceptance criteria** - focused tests pass"}`,
	})
	defer restore()

	triage := runKanbanJSONTask(t, "create", "rough release task", "--body", "needs concrete acceptance", "--triage", "--json")
	if triage.Status != kanban.StatusTriage {
		t.Fatalf("triage status = %q, want %q", triage.Status, kanban.StatusTriage)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "specify", triage.ID, "--author", "triage-bot", "--json")
	if err != nil {
		t.Fatalf("kanban specify --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action  string                `json:"action"`
		Outcome kanban.SpecifyOutcome `json:"outcome"`
		Task    kanban.Task           `json:"task"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("specify JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "specified" || !got.Outcome.OK {
		t.Fatalf("report = %+v, want successful specified action", got)
	}
	if got.Task.Status != kanban.StatusReady || got.Task.Title != "Specify release task" || !strings.Contains(got.Task.Body, "**Goal**") {
		t.Fatalf("specified task = %+v, want ready title/body from specifier", got.Task)
	}
}

func TestKanbanSpecifyCommandUnavailableIsTypedEvidence(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	restore := stubKanbanTriageSpecifier(nil)
	defer restore()

	triage := runKanbanJSONTask(t, "create", "rough task", "--triage", "--json")
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "specify", triage.ID)
	if err == nil {
		t.Fatalf("kanban specify without specifier error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "auxiliary client unavailable") {
		t.Fatalf("stderr = %q, want typed unavailable evidence", stderr)
	}
	got := runKanbanJSONTask(t, "show", triage.ID, "--json")
	if got.Status != kanban.StatusTriage {
		t.Fatalf("unavailable specify mutated task status = %q, want triage", got.Status)
	}
}

func TestKanbanCommandListJSONFiltersByStatus(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	ready := runKanbanJSONTask(t, "create", "Ready task", "--json")
	parent := runKanbanJSONTask(t, "create", "Parent", "--json")
	_ = runKanbanJSONTask(t, "create", "Todo child", "--parent", parent.ID, "--json")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "list", "--status", string(kanban.StatusReady), "--json")
	if err != nil {
		t.Fatalf("kanban list --json error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	// list --json wraps the array under `{build, tasks: [...]}` so
	// fleet automation aggregating Kanban inventory across machines
	// gets binary attribution alongside the entries — same convention
	// as the rest of the `--json` arc.
	var listGot struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Tasks []kanban.Task `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &listGot); err != nil {
		t.Fatalf("list JSON decode error = %v\nstdout=%s", err, stdout)
	}
	if listGot.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", listGot.Build.Version, Version)
	}
	var titles []string
	for _, task := range listGot.Tasks {
		if task.Status != kanban.StatusReady {
			t.Fatalf("filtered task status = %q, want only ready: %+v", task.Status, listGot.Tasks)
		}
		titles = append(titles, task.Title)
	}
	if !containsKanbanString(titles, ready.Title) || !containsKanbanString(titles, parent.Title) || containsKanbanString(titles, "Todo child") {
		t.Fatalf("ready titles = %v, want ready+parent and no todo child", titles)
	}
}

func TestKanbanTaskCommandsUseCurrentBoard(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	defaultTask := runKanbanJSONTask(t, "create", "default board task", "--json")
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "create", "alpha"); err != nil {
		t.Fatalf("kanban boards create alpha: %v\nstderr=%s", err, stderr)
	}
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "switch", "alpha"); err != nil {
		t.Fatalf("kanban boards switch alpha: %v\nstderr=%s", err, stderr)
	}

	alphaTask := runKanbanJSONTask(t, "create", "alpha board task", "--json")
	alphaTasks := runKanbanJSONTasks(t, "list", "--json")
	if !containsKanbanTaskTitle(alphaTasks, alphaTask.Title) {
		t.Fatalf("alpha board tasks = %+v, want %q", alphaTasks, alphaTask.Title)
	}
	if containsKanbanTaskTitle(alphaTasks, defaultTask.Title) {
		t.Fatalf("alpha board tasks = %+v, should not include default-board task %q", alphaTasks, defaultTask.Title)
	}

	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "switch", "default"); err != nil {
		t.Fatalf("kanban boards switch default: %v\nstderr=%s", err, stderr)
	}
	defaultTasks := runKanbanJSONTasks(t, "list", "--json")
	if !containsKanbanTaskTitle(defaultTasks, defaultTask.Title) {
		t.Fatalf("default board tasks = %+v, want %q", defaultTasks, defaultTask.Title)
	}
	if containsKanbanTaskTitle(defaultTasks, alphaTask.Title) {
		t.Fatalf("default board tasks = %+v, should not include alpha-board task %q", defaultTasks, alphaTask.Title)
	}
}

func TestKanbanCommandBoardFlagRoutesWithoutSwitchingCurrent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)

	defaultTask := runKanbanJSONTask(t, "create", "default current task", "--json")
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "create", "alpha"); err != nil {
		t.Fatalf("kanban boards create alpha: %v\nstderr=%s", err, stderr)
	}

	alphaTask := runKanbanJSONTask(t, "--board", "alpha", "create", "alpha one-off task", "--json")
	alphaTasks := runKanbanJSONTasks(t, "--board", "alpha", "list", "--json")
	if !containsKanbanTaskTitle(alphaTasks, alphaTask.Title) {
		t.Fatalf("--board alpha tasks = %+v, want %q", alphaTasks, alphaTask.Title)
	}
	if containsKanbanTaskTitle(alphaTasks, defaultTask.Title) {
		t.Fatalf("--board alpha tasks = %+v, should not include default-board task %q", alphaTasks, defaultTask.Title)
	}

	defaultTasks := runKanbanJSONTasks(t, "list", "--json")
	if !containsKanbanTaskTitle(defaultTasks, defaultTask.Title) {
		t.Fatalf("default/current board tasks = %+v, want %q", defaultTasks, defaultTask.Title)
	}
	if containsKanbanTaskTitle(defaultTasks, alphaTask.Title) {
		t.Fatalf("default/current board tasks = %+v, should not include --board alpha task %q", defaultTasks, alphaTask.Title)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "current", "--json")
	if err != nil {
		t.Fatalf("kanban boards current --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var current struct {
		Current string `json:"current"`
	}
	if err := json.Unmarshal([]byte(stdout), &current); err != nil {
		t.Fatalf("current JSON decode: %v\nstdout=%s", err, stdout)
	}
	if current.Current != "default" {
		t.Fatalf("current board = %q, want default after one-off --board", current.Current)
	}
}

func TestKanbanCommandBoardFlagRejectsMissingBoardWithoutMutation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "--board", "missing", "list", "--json")
	if err == nil {
		t.Fatalf("kanban --board missing list error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, `board "missing" does not exist`) {
		t.Fatalf("missing-board output missing not-found evidence:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	if _, err := os.Stat(filepath.Join(root, "kanban", "boards", "missing")); !os.IsNotExist(err) {
		t.Fatalf("--board missing created board directory, stat err = %v", err)
	}
}

func TestKanbanCommandPreservesExplicitDBPin(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "home"))

	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "create", "alpha"); err != nil {
		t.Fatalf("kanban boards create alpha: %v\nstderr=%s", err, stderr)
	}
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "boards", "switch", "alpha"); err != nil {
		t.Fatalf("kanban boards switch alpha: %v\nstderr=%s", err, stderr)
	}

	pinnedPath := filepath.Join(root, "pinned", "kanban.db")
	t.Setenv("GORMES_KANBAN_DB", pinnedPath)
	pinnedTask := runKanbanJSONTask(t, "create", "pinned db task", "--json")
	pinnedTasks := runKanbanJSONTasks(t, "list", "--json")
	if !containsKanbanTaskTitle(pinnedTasks, pinnedTask.Title) {
		t.Fatalf("pinned DB tasks = %+v, want %q", pinnedTasks, pinnedTask.Title)
	}
	if _, err := os.Stat(pinnedPath); err != nil {
		t.Fatalf("explicit GORMES_KANBAN_DB path was not used: %v", err)
	}

	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "--board", "alpha", "create", "still pinned", "--json"); err != nil {
		t.Fatalf("kanban --board alpha create with pinned DB: %v\nstderr=%s", err, stderr)
	}
	pinnedTasks = runKanbanJSONTasks(t, "--board", "alpha", "list", "--json")
	if !containsKanbanTaskTitle(pinnedTasks, "still pinned") {
		t.Fatalf("pinned DB tasks with --board alpha = %+v, want still pinned task", pinnedTasks)
	}
}

// TestKanbanInitCommand_JSONEmitsStructuredReport proves
// `gormes kanban init --json` emits a parseable
// `{build, action, path}` document so fleet automation provisioning
// the local Kanban database across machines can verify the seed
// outcome with binary attribution. The default text output ("kanban
// initialized at <path>") remains unchanged for shell consumers.
func TestKanbanInitCommand_JSONEmitsStructuredReport(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "init", "--json")
	if err != nil {
		t.Fatalf("kanban init --json: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		Path   string `json:"path"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "initialized" {
		t.Errorf("action = %q, want initialized", got.Action)
	}
	if got.Path == "" {
		t.Errorf("path empty, want kanban.db path")
	}
}

// TestKanbanClaimCommand_JSONIncludesBuildProvenance proves
// `gormes kanban claim <id> --json` emits a top-level `build` envelope
// so fleet automation orchestrating worker assignment across machines
// can attribute each claim outcome to the binary version that emitted
// it. Existing `task`/`claimed` fields stay addressable.
func TestKanbanClaimCommand_JSONIncludesBuildProvenance(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	created := runKanbanJSONTask(t, "create", "Claimable task", "--json")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "claim", created.ID, "--worker", "test-worker", "--json")
	if err != nil {
		t.Fatalf("kanban claim --json: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
		Claimed bool `json:"claimed"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("claim JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Task.ID != created.ID {
		t.Errorf("task.id = %q, want %q (still addressable)", got.Task.ID, created.ID)
	}
	if !got.Claimed {
		t.Errorf("claimed = false, want true")
	}
}

// TestKanbanCommand_LifecycleVerbsEmitJSON proves
// `gormes kanban {complete,block,unblock,link} ... --json` emit a parseable
// `{build, action, id, ...}` document so fleet automation orchestrating
// Kanban state across machines can observe outcomes without scraping the
// "Completed X / Blocked X / Unblocked X / Linked X -> Y" prose. Same
// `{build, action}` lead as the rest of the `--json` arc.
func TestKanbanCommand_LifecycleVerbsEmitJSON(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	parent := runKanbanJSONTask(t, "create", "Parent", "--json")
	child := runKanbanJSONTask(t, "create", "Child", "--json")

	type lifecycle struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		ID     string `json:"id,omitempty"`
		Parent string `json:"parent,omitempty"`
		Child  string `json:"child,omitempty"`
		Reason string `json:"reason,omitempty"`
	}

	t.Run("block", func(t *testing.T) {
		stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "block", child.ID, "waiting on parent", "--json")
		if err != nil {
			t.Fatalf("kanban block --json: %v\nstderr=%s", err, stderr)
		}
		var got lifecycle
		if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
			t.Fatalf("block JSON decode: %v\nstdout=%s", jsonErr, stdout)
		}
		if got.Build.Version != Version {
			t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
		}
		if got.Action != "blocked" {
			t.Errorf("action = %q, want blocked", got.Action)
		}
		if got.ID != child.ID {
			t.Errorf("id = %q, want %q", got.ID, child.ID)
		}
		if got.Reason != "waiting on parent" {
			t.Errorf("reason = %q, want %q", got.Reason, "waiting on parent")
		}
	})

	t.Run("unblock", func(t *testing.T) {
		stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "unblock", child.ID, "--json")
		if err != nil {
			t.Fatalf("kanban unblock --json: %v\nstderr=%s", err, stderr)
		}
		var got lifecycle
		if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
			t.Fatalf("unblock JSON decode: %v\nstdout=%s", jsonErr, stdout)
		}
		if got.Action != "unblocked" {
			t.Errorf("action = %q, want unblocked", got.Action)
		}
		if got.ID != child.ID {
			t.Errorf("id = %q, want %q", got.ID, child.ID)
		}
	})

	t.Run("link", func(t *testing.T) {
		stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "link", parent.ID, child.ID, "--json")
		if err != nil {
			t.Fatalf("kanban link --json: %v\nstderr=%s", err, stderr)
		}
		var got lifecycle
		if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
			t.Fatalf("link JSON decode: %v\nstdout=%s", jsonErr, stdout)
		}
		if got.Action != "linked" {
			t.Errorf("action = %q, want linked", got.Action)
		}
		if got.Parent != parent.ID {
			t.Errorf("parent = %q, want %q", got.Parent, parent.ID)
		}
		if got.Child != child.ID {
			t.Errorf("child = %q, want %q", got.Child, child.ID)
		}
	})

	t.Run("complete", func(t *testing.T) {
		stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "complete", parent.ID, "--result", "ready", "--json")
		if err != nil {
			t.Fatalf("kanban complete --json: %v\nstderr=%s", err, stderr)
		}
		var got lifecycle
		if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
			t.Fatalf("complete JSON decode: %v\nstdout=%s", jsonErr, stdout)
		}
		if got.Action != "completed" {
			t.Errorf("action = %q, want completed", got.Action)
		}
		if got.ID != parent.ID {
			t.Errorf("id = %q, want %q", got.ID, parent.ID)
		}
	})
}

// TestKanbanSingleTaskCommands_JSONIncludeBuildProvenance proves
// `gormes kanban create --json` and `gormes kanban show <id> --json`
// emit a top-level `build` envelope so fleet automation orchestrating
// Kanban state across machines can attribute each task document to the
// binary version that emitted it. Existing kanban.Task fields stay
// addressable through struct embedding — additive change.
func TestKanbanSingleTaskCommands_JSONIncludeBuildProvenance(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "create", "Build attribution test", "--json")
	if err != nil {
		t.Fatalf("kanban create --json: %v\nstderr=%s", err, stderr)
	}
	var createGot struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &createGot); jsonErr != nil {
		t.Fatalf("create JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if createGot.Build.Version != Version {
		t.Errorf("create build.version = %q, want %q", createGot.Build.Version, Version)
	}
	if createGot.Title != "Build attribution test" {
		t.Errorf("create title = %q, want it (still addressable)", createGot.Title)
	}
	if createGot.ID == "" {
		t.Errorf("create id empty, want populated (still addressable)")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "show", createGot.ID, "--json")
	if err != nil {
		t.Fatalf("kanban show --json: %v\nstderr=%s", err, stderr)
	}
	var showGot struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		ID string `json:"id"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &showGot); jsonErr != nil {
		t.Fatalf("show JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if showGot.Build.Version != Version {
		t.Errorf("show build.version = %q, want %q", showGot.Build.Version, Version)
	}
	if showGot.ID != createGot.ID {
		t.Errorf("show id = %q, want %q (still addressable)", showGot.ID, createGot.ID)
	}
}

func runKanbanJSONTask(t *testing.T, args ...string) kanban.Task {
	t.Helper()
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), append([]string{"kanban"}, args...)...)
	if err != nil {
		t.Fatalf("gormes kanban %v error = %v\nstdout=%s\nstderr=%s", args, err, stdout, stderr)
	}
	var task kanban.Task
	if err := json.Unmarshal([]byte(stdout), &task); err != nil {
		t.Fatalf("gormes kanban %v JSON decode error = %v\nstdout=%s", args, err, stdout)
	}
	return task
}

func runKanbanJSONTasks(t *testing.T, args ...string) []kanban.Task {
	t.Helper()
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), append([]string{"kanban"}, args...)...)
	if err != nil {
		t.Fatalf("gormes kanban %v error = %v\nstdout=%s\nstderr=%s", args, err, stdout, stderr)
	}
	var list struct {
		Tasks []kanban.Task `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("gormes kanban %v JSON decode error = %v\nstdout=%s", args, err, stdout)
	}
	return list.Tasks
}

func containsKanbanTaskTitle(tasks []kanban.Task, title string) bool {
	for _, task := range tasks {
		if task.Title == title {
			return true
		}
	}
	return false
}

func containsKanbanString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type recordingCommandTriageSpecifier struct {
	response string
	request  kanban.TriageSpecRequest
}

func (s *recordingCommandTriageSpecifier) CompleteTriageSpec(_ context.Context, req kanban.TriageSpecRequest) (string, error) {
	s.request = req
	return s.response, nil
}

func stubKanbanTriageSpecifier(specifier kanban.TriageSpecifier) func() {
	previous := newKanbanTriageSpecifier
	newKanbanTriageSpecifier = func(config.Config) (kanban.TriageSpecifier, error) {
		return specifier, nil
	}
	return func() {
		newKanbanTriageSpecifier = previous
	}
}
