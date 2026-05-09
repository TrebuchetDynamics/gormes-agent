package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

type fakeKanbanStore struct {
	tasks   []kanban.Task
	listErr error
	getErr  error
}

type fakeKanbanDispatcher struct {
	result kanban.DispatchResult
	err    error
	calls  int
	max    int
}

func (f *fakeKanbanDispatcher) DispatchKanban(_ context.Context, opts KanbanDispatchOptions) (kanban.DispatchResult, error) {
	f.calls++
	f.max = opts.MaxSpawn
	return f.result, f.err
}

func (f *fakeKanbanStore) ListTasks(_ context.Context, filter kanban.ListFilter) ([]kanban.Task, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	result := make([]kanban.Task, 0, len(f.tasks))
	for _, t := range f.tasks {
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.Assignee != "" && t.Assignee != filter.Assignee {
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

func (f *fakeKanbanStore) GetTask(_ context.Context, id string) (kanban.Task, error) {
	if f.getErr != nil {
		return kanban.Task{}, f.getErr
	}
	for _, t := range f.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return kanban.Task{}, errors.New("task not found")
}

func TestDashboardKanban_Unauthorized(t *testing.T) {
	srv := NewServer(Config{
		DashboardSessionToken: "fixture-token",
		KanbanStore:           &fakeKanbanStore{},
	})
	h := srv.Handler()

	for _, path := range []string{"/api/kanban", "/api/kanban/tasks", "/api/kanban/tasks/test-1"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401; body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestDashboardKanban_StoreUnavailable(t *testing.T) {
	srv := NewServer(Config{
		DashboardSessionToken: "fixture-token",
		KanbanStore:           nil,
	})
	h := srv.Handler()
	auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	for _, path := range []string{"/api/kanban", "/api/kanban/tasks", "/api/kanban/tasks/test-1"} {
		rec := getJSON(t, h, path, auth)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status = %d, want 503; body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestDashboardKanban_PanelInStatus(t *testing.T) {
	t.Run("panel enabled when store is configured", func(t *testing.T) {
		srv := NewServer(Config{
			DashboardSessionToken: "fixture-token",
			KanbanStore:           &fakeKanbanStore{},
		})
		h := srv.Handler()
		auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

		rec := getJSON(t, h, "/api/status", auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var status struct {
			Panels map[string]struct {
				State     string   `json:"state"`
				Endpoints []string `json:"endpoints,omitempty"`
			} `json:"panels"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
			t.Fatalf("decode status: %v", err)
		}
		panel, ok := status.Panels["kanban"]
		if !ok {
			t.Fatal("kanban panel missing from dashboard status")
		}
		if panel.State != "enabled" {
			t.Fatalf("kanban panel state = %q, want enabled", panel.State)
		}
	})

	t.Run("panel disabled when store is nil", func(t *testing.T) {
		srv := NewServer(Config{
			DashboardSessionToken: "fixture-token",
			KanbanStore:           nil,
		})
		h := srv.Handler()
		auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

		rec := getJSON(t, h, "/api/status", auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var status struct {
			Panels map[string]struct {
				State  string `json:"state"`
				Reason string `json:"reason,omitempty"`
			} `json:"panels"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
			t.Fatalf("decode status: %v", err)
		}
		panel, ok := status.Panels["kanban"]
		if !ok {
			t.Fatal("kanban panel missing from dashboard status")
		}
		if panel.State != "disabled" {
			t.Fatalf("kanban panel state = %q, want disabled", panel.State)
		}
	})
}

// TestDashboardKanban_BuildAttribution proves the dashboard kanban
// endpoints (`/api/kanban`, `/api/kanban/tasks`, `/api/kanban/tasks/{id}`)
// each carry the configured BuildInfo at the top of the JSON response
// so fleet automation aggregating Kanban dashboard state across
// machines can attribute each response to the binary version that
// emitted it. Same convention as `/api/status` (slice 110) and the
// rest of the `--json` arc. Existing top-level fields stay
// addressable through struct embedding for the typed responses.
func TestDashboardKanban_BuildAttribution(t *testing.T) {
	store := &fakeKanbanStore{
		tasks: []kanban.Task{
			{ID: "t1", Title: "task one", Status: kanban.StatusTriage},
		},
	}
	srv := NewServer(Config{
		DashboardSessionToken: "fixture-token",
		KanbanStore:           store,
		BuildInfo: BuildInfo{
			Version:   "test-version-9.9.9",
			GitCommit: "feedface",
			GitDirty:  false,
			GoVersion: "go1.23.0-test",
		},
	})
	h := srv.Handler()
	auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	for _, path := range []string{"/api/kanban", "/api/kanban/tasks", "/api/kanban/tasks/t1"} {
		rec := getJSON(t, h, path, auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d; body=%s", path, rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
			} `json:"build"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: decode: %v\nbody=%s", path, err, rec.Body.String())
		}
		if got.Build.Version != "test-version-9.9.9" {
			t.Errorf("%s: build.version = %q, want test-version-9.9.9 (body=%s)", path, got.Build.Version, rec.Body.String())
		}
		if got.Build.GitCommit != "feedface" {
			t.Errorf("%s: build.git_commit = %q, want feedface", path, got.Build.GitCommit)
		}
	}
}

func TestDashboardKanban_OverviewLaneSummary(t *testing.T) {
	store := &fakeKanbanStore{
		tasks: []kanban.Task{
			{ID: "t1", Title: "fix bug", Status: kanban.StatusTriage, Priority: 2},
			{ID: "t2", Title: "add feature", Status: kanban.StatusTodo, Priority: 5},
			{ID: "t3", Title: "review", Status: kanban.StatusDone, Priority: 1},
			{ID: "t4", Title: "urgent fix", Status: kanban.StatusTriage, Priority: 8},
		},
	}
	srv := NewServer(Config{
		DashboardSessionToken: "fixture-token",
		KanbanStore:           store,
	})
	h := srv.Handler()
	auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	rec := getJSON(t, h, "/api/kanban", auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got DashboardKanbanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode kanban: %v", err)
	}
	if got.TotalTasks != 4 {
		t.Fatalf("total_tasks = %d, want 4", got.TotalTasks)
	}
	lanesByStatus := map[string]int{}
	for _, lane := range got.Lanes {
		lanesByStatus[lane.Status] = lane.Count
	}
	if lanesByStatus[string(kanban.StatusTriage)] != 2 {
		t.Errorf("triage count = %d, want 2", lanesByStatus[string(kanban.StatusTriage)])
	}
	if lanesByStatus[string(kanban.StatusTodo)] != 1 {
		t.Errorf("todo count = %d, want 1", lanesByStatus[string(kanban.StatusTodo)])
	}
	if lanesByStatus[string(kanban.StatusDone)] != 1 {
		t.Errorf("done count = %d, want 1", lanesByStatus[string(kanban.StatusDone)])
	}
	if got.Dispatcher.Available {
		t.Error("dispatcher should not be available in dashboard kanban view")
	}
}

func TestDashboardKanban_TaskListWithFilters(t *testing.T) {
	store := &fakeKanbanStore{
		tasks: []kanban.Task{
			{ID: "t1", Title: "fix bug", Status: kanban.StatusTriage, Assignee: "alice", Priority: 2},
			{ID: "t2", Title: "add feature", Status: kanban.StatusTodo, Assignee: "bob", Priority: 5},
			{ID: "t3", Title: "review", Status: kanban.StatusDone, Assignee: "alice", Priority: 1},
		},
	}
	srv := NewServer(Config{
		DashboardSessionToken: "fixture-token",
		KanbanStore:           store,
	})
	h := srv.Handler()
	auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	t.Run("list all tasks", func(t *testing.T) {
		rec := getJSON(t, h, "/api/kanban/tasks", auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var got struct {
			Tasks []kanban.Task `json:"tasks"`
			Total int           `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode tasks: %v", err)
		}
		if got.Total != 3 {
			t.Fatalf("total = %d, want 3", got.Total)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		rec := getJSON(t, h, "/api/kanban/tasks?status=triage", auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var got struct {
			Tasks []kanban.Task `json:"tasks"`
			Total int           `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode tasks: %v", err)
		}
		if got.Total != 1 {
			t.Fatalf("total = %d, want 1", got.Total)
		}
		if got.Tasks[0].ID != "t1" {
			t.Errorf("task id = %q, want t1", got.Tasks[0].ID)
		}
	})

	t.Run("filter by assignee", func(t *testing.T) {
		rec := getJSON(t, h, "/api/kanban/tasks?assignee=alice", auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var got struct {
			Tasks []kanban.Task `json:"tasks"`
			Total int           `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode tasks: %v", err)
		}
		if got.Total != 2 {
			t.Fatalf("total = %d, want 2", got.Total)
		}
	})
}

func TestDashboardKanban_TaskByID(t *testing.T) {
	store := &fakeKanbanStore{
		tasks: []kanban.Task{
			{ID: "t1", Title: "fix bug", Status: kanban.StatusTriage, Priority: 2, Body: "needs attention"},
		},
	}
	srv := NewServer(Config{
		DashboardSessionToken: "fixture-token",
		KanbanStore:           store,
	})
	h := srv.Handler()
	auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	t.Run("get existing task", func(t *testing.T) {
		rec := getJSON(t, h, "/api/kanban/tasks/t1", auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var got kanban.Task
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode task: %v", err)
		}
		if got.ID != "t1" {
			t.Errorf("task id = %q, want t1", got.ID)
		}
		if got.Body != "needs attention" {
			t.Errorf("body = %q, want 'needs attention'", got.Body)
		}
	})

	t.Run("get missing task", func(t *testing.T) {
		rec := getJSON(t, h, "/api/kanban/tasks/nonexistent", auth)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestDashboardKanban_DispatchQuickPath(t *testing.T) {
	auth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	t.Run("requires dashboard session token", func(t *testing.T) {
		dispatcher := &fakeKanbanDispatcher{}
		srv := NewServer(Config{
			DashboardSessionToken: "fixture-token",
			KanbanDispatcher:      dispatcher,
		})
		rec := postJSON(t, srv.Handler(), "/api/kanban/dispatch?max=2", nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
		}
		if dispatcher.calls != 0 {
			t.Fatalf("dispatcher calls = %d, want 0 for unauthorized request", dispatcher.calls)
		}
	})

	t.Run("runs injected dispatcher with max and build attribution", func(t *testing.T) {
		dispatcher := &fakeKanbanDispatcher{
			result: kanban.DispatchResult{
				ReclaimedIDs:       []string{"old-claim"},
				Spawned:            []kanban.SpawnRecord{{TaskID: "t1", Assignee: "alice", WorkspacePath: "/tmp/work", PID: 42}},
				SkippedUnassigned:  []string{"t2"},
				SpawnFailedIDs:     []string{"t3"},
				AutoBlockedTaskIDs: []string{"t4"},
			},
		}
		srv := NewServer(Config{
			DashboardSessionToken: "fixture-token",
			KanbanDispatcher:      dispatcher,
			BuildInfo: BuildInfo{
				Version:   "test-version-9.9.9",
				GitCommit: "feedface",
			},
		})

		rec := postJSON(t, srv.Handler(), "/api/kanban/dispatch?max=2", nil, auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
		if dispatcher.calls != 1 {
			t.Fatalf("dispatcher calls = %d, want 1", dispatcher.calls)
		}
		if dispatcher.max != 2 {
			t.Fatalf("dispatcher max = %d, want 2", dispatcher.max)
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
			} `json:"build"`
			Result kanban.DispatchResult `json:"result"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode dispatch response: %v\nbody=%s", err, rec.Body.String())
		}
		if got.Build.Version != "test-version-9.9.9" || got.Build.GitCommit != "feedface" {
			t.Fatalf("build = %+v, want version and commit attribution", got.Build)
		}
		if len(got.Result.Spawned) != 1 || got.Result.Spawned[0].TaskID != "t1" {
			t.Fatalf("result.spawned = %+v, want task t1", got.Result.Spawned)
		}
		if len(got.Result.ReclaimedIDs) != 1 || got.Result.ReclaimedIDs[0] != "old-claim" {
			t.Fatalf("result.reclaimed_ids = %+v, want old-claim", got.Result.ReclaimedIDs)
		}
	})

	t.Run("reports unavailable seam", func(t *testing.T) {
		srv := NewServer(Config{DashboardSessionToken: "fixture-token"})
		rec := postJSON(t, srv.Handler(), "/api/kanban/dispatch", nil, auth)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "kanban_dispatcher_unavailable") {
			t.Fatalf("body missing kanban_dispatcher_unavailable: %s", rec.Body.String())
		}
	})

	t.Run("reports dispatch failure", func(t *testing.T) {
		dispatcher := &fakeKanbanDispatcher{err: errors.New("worker_spawn_failed: denied")}
		srv := NewServer(Config{
			DashboardSessionToken: "fixture-token",
			KanbanDispatcher:      dispatcher,
		})
		rec := postJSON(t, srv.Handler(), "/api/kanban/dispatch", nil, auth)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "kanban_dispatch_failed") {
			t.Fatalf("body missing kanban_dispatch_failed: %s", rec.Body.String())
		}
	})
}
