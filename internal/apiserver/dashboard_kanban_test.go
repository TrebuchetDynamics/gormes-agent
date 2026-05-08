package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

type fakeKanbanStore struct {
	tasks   []kanban.Task
	listErr error
	getErr  error
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
