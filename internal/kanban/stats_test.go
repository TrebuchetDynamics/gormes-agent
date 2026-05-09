package kanban

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBoardStatsCountsNonArchivedTasks(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 5, 9, 18, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now.Add(-20 * time.Minute) }
	readyOld, err := store.CreateTask(ctx, CreateTaskInput{Title: "old ready", Assignee: "alice"})
	if err != nil {
		t.Fatalf("CreateTask(old ready) error = %v", err)
	}
	store.now = func() time.Time { return now.Add(-15 * time.Minute) }
	if _, err := store.CreateTask(ctx, CreateTaskInput{Title: "triage", Assignee: "alice", Triage: true}); err != nil {
		t.Fatalf("CreateTask(triage) error = %v", err)
	}
	store.now = func() time.Time { return now.Add(-12 * time.Minute) }
	if _, err := store.CreateTask(ctx, CreateTaskInput{Title: "new ready", Assignee: "bob"}); err != nil {
		t.Fatalf("CreateTask(new ready) error = %v", err)
	}
	parent, err := store.CreateTask(ctx, CreateTaskInput{Title: "parent"})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	if _, err := store.CreateTask(ctx, CreateTaskInput{Title: "todo child", Assignee: "alice", ParentIDs: []string{parent.ID}}); err != nil {
		t.Fatalf("CreateTask(todo child) error = %v", err)
	}
	blocked, err := store.CreateTask(ctx, CreateTaskInput{Title: "blocked", Assignee: "bob"})
	if err != nil {
		t.Fatalf("CreateTask(blocked) error = %v", err)
	}
	if err := store.BlockTask(ctx, blocked.ID, BlockTaskInput{Reason: "waiting"}); err != nil {
		t.Fatalf("BlockTask() error = %v", err)
	}
	running, err := store.CreateTask(ctx, CreateTaskInput{Title: "running", Assignee: "alice"})
	if err != nil {
		t.Fatalf("CreateTask(running) error = %v", err)
	}
	if _, claimed, err := store.ClaimTask(ctx, running.ID, ClaimTaskInput{Worker: "worker"}); err != nil || !claimed {
		t.Fatalf("ClaimTask() claimed=%v err=%v", claimed, err)
	}
	done, err := store.CreateTask(ctx, CreateTaskInput{Title: "done", Assignee: "bob"})
	if err != nil {
		t.Fatalf("CreateTask(done) error = %v", err)
	}
	if err := store.CompleteTask(ctx, done.ID, CompleteTaskInput{Result: "finished"}); err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}
	archived, err := store.CreateTask(ctx, CreateTaskInput{Title: "archived", Assignee: "alice"})
	if err != nil {
		t.Fatalf("CreateTask(archived) error = %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, string(StatusArchived), archived.ID); err != nil {
		t.Fatalf("archive fixture task: %v", err)
	}

	store.now = func() time.Time { return now }
	stats, err := store.BoardStats(ctx)
	if err != nil {
		t.Fatalf("BoardStats() error = %v", err)
	}

	wantStatus := map[string]int{
		string(StatusTriage):  1,
		string(StatusTodo):    1,
		string(StatusReady):   3,
		string(StatusRunning): 1,
		string(StatusBlocked): 1,
		string(StatusDone):    1,
	}
	for status, want := range wantStatus {
		if got := stats.ByStatus[status]; got != want {
			t.Errorf("ByStatus[%s] = %d, want %d", status, got, want)
		}
	}
	if _, ok := stats.ByStatus[string(StatusArchived)]; ok {
		t.Fatalf("archived task %s was included in status stats: %+v", archived.ID, stats.ByStatus)
	}
	if got := stats.ByAssignee["alice"][string(StatusReady)]; got != 1 {
		t.Errorf("alice ready count = %d, want 1 (old ready %s)", got, readyOld.ID)
	}
	if got := stats.ByAssignee["alice"][string(StatusTodo)]; got != 1 {
		t.Errorf("alice todo count = %d, want 1", got)
	}
	if got := stats.ByAssignee["alice"][string(StatusRunning)]; got != 1 {
		t.Errorf("alice running count = %d, want 1", got)
	}
	if got := stats.ByAssignee["bob"][string(StatusBlocked)]; got != 1 {
		t.Errorf("bob blocked count = %d, want 1", got)
	}
	if got := stats.ByAssignee["bob"][string(StatusDone)]; got != 1 {
		t.Errorf("bob done count = %d, want 1", got)
	}
	if stats.OldestReadyAgeSeconds == nil || *stats.OldestReadyAgeSeconds != 1200 {
		t.Fatalf("OldestReadyAgeSeconds = %v, want 1200", stats.OldestReadyAgeSeconds)
	}
	if stats.Now != now.Unix() {
		t.Fatalf("Now = %d, want %d", stats.Now, now.Unix())
	}
}

func TestBoardStatsEmptyBoardUsesEmptyObjectsAndNullAge(t *testing.T) {
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	stats, err := store.BoardStats(context.Background())
	if err != nil {
		t.Fatalf("BoardStats() error = %v", err)
	}
	if stats.ByStatus == nil || len(stats.ByStatus) != 0 {
		t.Fatalf("ByStatus = %#v, want empty non-nil map", stats.ByStatus)
	}
	if stats.ByAssignee == nil || len(stats.ByAssignee) != 0 {
		t.Fatalf("ByAssignee = %#v, want empty non-nil map", stats.ByAssignee)
	}
	if stats.OldestReadyAgeSeconds != nil {
		t.Fatalf("OldestReadyAgeSeconds = %v, want nil", *stats.OldestReadyAgeSeconds)
	}
}

func TestBoardStatsDoesNotTouchHermesHome(t *testing.T) {
	hermesHome := filepath.Join(t.TempDir(), "hermes")
	t.Setenv("HERMES_HOME", hermesHome)
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	if _, err := store.BoardStats(context.Background()); err != nil {
		t.Fatalf("BoardStats() error = %v", err)
	}
	if _, err := os.Stat(hermesHome); !os.IsNotExist(err) {
		t.Fatalf("BoardStats touched HERMES_HOME, stat err=%v", err)
	}
}
