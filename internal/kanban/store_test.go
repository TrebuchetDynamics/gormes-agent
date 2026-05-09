package kanban

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestKanbanStorePromotesChildAfterParentCompletes(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	parent, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Design schema",
		Assignee: "researcher",
	})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	if parent.Status != StatusReady {
		t.Fatalf("parent.Status = %q, want %q", parent.Status, StatusReady)
	}

	child, err := store.CreateTask(ctx, CreateTaskInput{
		Title:     "Implement API",
		Assignee:  "backend-dev",
		ParentIDs: []string{parent.ID},
	})
	if err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}
	if child.Status != StatusTodo {
		t.Fatalf("child.Status = %q, want %q", child.Status, StatusTodo)
	}

	if err := store.CompleteTask(ctx, parent.ID, CompleteTaskInput{Result: "schema ready"}); err != nil {
		t.Fatalf("CompleteTask(parent) error = %v", err)
	}

	got, err := store.GetTask(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetTask(child) error = %v", err)
	}
	if got.Status != StatusReady {
		t.Fatalf("child after parent complete Status = %q, want %q", got.Status, StatusReady)
	}
}

func TestKanbanStoreAddColumnIfMissingIgnoresDuplicateColumnRace(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if err := store.addColumnIfMissing(ctx, "tasks", "spawn_failures", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		t.Fatalf("addColumnIfMissing duplicate column error = %v, want nil", err)
	}

	if err := store.addColumnIfMissing(ctx, "tasks", "race_optional", "TEXT NOT NULL DEFAULT ''"); err != nil {
		t.Fatalf("addColumnIfMissing new column error = %v", err)
	}
	rows, err := store.db.QueryContext(ctx, `SELECT race_optional FROM tasks LIMIT 0`)
	if err != nil {
		t.Fatalf("race_optional column not queryable after migration: %v", err)
	}
	rows.Close()

	if err := store.addColumnIfMissing(ctx, "tasks", "race_optional", "TEXT NOT NULL DEFAULT ''"); err != nil {
		t.Fatalf("addColumnIfMissing repeated duplicate error = %v, want nil", err)
	}
	err = store.ensureColumn(ctx, "missing_table", "race_optional", "TEXT")
	if err == nil {
		t.Fatal("addColumnIfMissing missing table error = nil, want non-duplicate migration failure")
	}
	if !strings.Contains(err.Error(), "migrate kanban missing_table.race_optional") {
		t.Fatalf("missing table error = %v, want table/column migrate evidence", err)
	}
}

func TestKanbanStoreConcurrentClaimOnlyOneWinner(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "Write tests"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan bool, 2)
	for _, worker := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(worker string) {
			defer wg.Done()
			_, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{
				Worker: worker,
				TTL:    time.Minute,
			})
			if err != nil {
				t.Errorf("ClaimTask(%s) error = %v", worker, err)
				return
			}
			results <- claimed
		}(worker)
	}
	wg.Wait()
	close(results)

	var winners int
	for claimed := range results {
		if claimed {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("claimed winners = %d, want 1", winners)
	}

	got, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusRunning {
		t.Fatalf("Status after claim = %q, want %q", got.Status, StatusRunning)
	}
	if got.ClaimLock == "" || got.ClaimExpires.IsZero() {
		t.Fatalf("claim evidence missing: %+v", got)
	}
}

func TestKanbanStoreLinkRejectsCyclesAndDemotesReadyChildren(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	parent, err := store.CreateTask(ctx, CreateTaskInput{Title: "Design schema"})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	child, err := store.CreateTask(ctx, CreateTaskInput{Title: "Implement API"})
	if err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}
	if child.Status != StatusReady {
		t.Fatalf("child before link Status = %q, want %q", child.Status, StatusReady)
	}

	if err := store.LinkTasks(ctx, parent.ID, child.ID); err != nil {
		t.Fatalf("LinkTasks(parent, child) error = %v", err)
	}
	got, err := store.GetTask(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetTask(child) error = %v", err)
	}
	if got.Status != StatusTodo {
		t.Fatalf("child after incomplete parent link Status = %q, want %q", got.Status, StatusTodo)
	}

	if err := store.LinkTasks(ctx, child.ID, parent.ID); err == nil {
		t.Fatal("LinkTasks(child, parent) error = nil, want cycle rejection")
	}
}

func TestKanbanStoreBlockUnblockRecomputesReadiness(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	parent, err := store.CreateTask(ctx, CreateTaskInput{Title: "Design schema"})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	child, err := store.CreateTask(ctx, CreateTaskInput{
		Title:     "Implement API",
		ParentIDs: []string{parent.ID},
	})
	if err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}
	if child.Status != StatusTodo {
		t.Fatalf("child initial Status = %q, want %q", child.Status, StatusTodo)
	}

	if err := store.BlockTask(ctx, child.ID, BlockTaskInput{Reason: "waiting for review"}); err != nil {
		t.Fatalf("BlockTask(child) error = %v", err)
	}
	blocked, err := store.GetTask(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetTask(blocked child) error = %v", err)
	}
	if blocked.Status != StatusBlocked || blocked.Result != "waiting for review" {
		t.Fatalf("blocked child = %+v, want blocked with reason", blocked)
	}

	if err := store.UnblockTask(ctx, child.ID); err != nil {
		t.Fatalf("UnblockTask(child) error = %v", err)
	}
	unblocked, err := store.GetTask(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetTask(unblocked child) error = %v", err)
	}
	if unblocked.Status != StatusTodo {
		t.Fatalf("unblocked child with incomplete parent Status = %q, want %q", unblocked.Status, StatusTodo)
	}

	if err := store.CompleteTask(ctx, parent.ID, CompleteTaskInput{Result: "done"}); err != nil {
		t.Fatalf("CompleteTask(parent) error = %v", err)
	}
	ready, err := store.GetTask(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetTask(child after parent complete) error = %v", err)
	}
	if ready.Status != StatusReady {
		t.Fatalf("child after parent complete Status = %q, want %q", ready.Status, StatusReady)
	}
}

func TestKanbanGC_PrunesOnlyTerminalTaskEvents(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	active, err := store.CreateTask(ctx, CreateTaskInput{Title: "active task"})
	if err != nil {
		t.Fatalf("CreateTask(active) error = %v", err)
	}
	done, err := store.CreateTask(ctx, CreateTaskInput{Title: "done task"})
	if err != nil {
		t.Fatalf("CreateTask(done) error = %v", err)
	}
	if err := store.CompleteTask(ctx, done.ID, CompleteTaskInput{Result: "done"}); err != nil {
		t.Fatalf("CompleteTask(done) error = %v", err)
	}
	archived, err := store.CreateTask(ctx, CreateTaskInput{Title: "archived task"})
	if err != nil {
		t.Fatalf("CreateTask(archived) error = %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, string(StatusArchived), archived.ID); err != nil {
		t.Fatalf("archive fixture task: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `DELETE FROM task_events`); err != nil {
		t.Fatalf("reset fixture events: %v", err)
	}

	old := now.Add(-45 * 24 * time.Hour)
	fresh := now.Add(-5 * 24 * time.Hour)
	insertKanbanEventFixture(t, store, active.ID, "active-old", old)
	insertKanbanEventFixture(t, store, done.ID, "done-old", old)
	insertKanbanEventFixture(t, store, done.ID, "done-fresh", fresh)
	insertKanbanEventFixture(t, store, archived.ID, "archived-old", old)
	insertKanbanEventFixture(t, store, archived.ID, "archived-fresh", fresh)

	deleted, err := store.PruneTerminalEvents(ctx, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("PruneTerminalEvents() error = %v", err)
	}
	if deleted != 2 {
		t.Fatalf("PruneTerminalEvents() deleted = %d, want 2", deleted)
	}

	activeEvents, err := store.ListEvents(ctx, active.ID)
	if err != nil {
		t.Fatalf("ListEvents(active) error = %v", err)
	}
	if len(activeEvents) != 1 || activeEvents[0].Kind != "active-old" {
		t.Fatalf("active events = %+v, want old non-terminal event preserved", activeEvents)
	}
	doneEvents, err := store.ListEvents(ctx, done.ID)
	if err != nil {
		t.Fatalf("ListEvents(done) error = %v", err)
	}
	if len(doneEvents) != 1 || doneEvents[0].Kind != "done-fresh" {
		t.Fatalf("done events = %+v, want only fresh terminal event", doneEvents)
	}
	archivedEvents, err := store.ListEvents(ctx, archived.ID)
	if err != nil {
		t.Fatalf("ListEvents(archived) error = %v", err)
	}
	if len(archivedEvents) != 1 || archivedEvents[0].Kind != "archived-fresh" {
		t.Fatalf("archived events = %+v, want only fresh terminal event", archivedEvents)
	}
}

func TestKanbanGC_PrunesOnlySelectedBoardLogFiles(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "kanban", "boards", "alpha", "kanban.db")
	store, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	alphaLogRoot := filepath.Join(root, "kanban", "boards", "alpha", "logs")
	betaLogRoot := filepath.Join(root, "kanban", "boards", "beta", "logs")
	oldAlpha := writeKanbanLogFixture(t, alphaLogRoot, "old-alpha.log", now.Add(-45*24*time.Hour))
	freshAlpha := writeKanbanLogFixture(t, alphaLogRoot, "fresh-alpha.log", now.Add(-5*24*time.Hour))
	oldBeta := writeKanbanLogFixture(t, betaLogRoot, "old-beta.log", now.Add(-45*24*time.Hour))
	keptDir := filepath.Join(alphaLogRoot, "worker-dir")
	if err := os.MkdirAll(keptDir, 0o755); err != nil {
		t.Fatalf("create log fixture dir: %v", err)
	}

	deleted, err := store.PruneWorkerLogs(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("PruneWorkerLogs() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneWorkerLogs() deleted = %d, want 1", deleted)
	}
	if _, err := os.Stat(oldAlpha); !os.IsNotExist(err) {
		t.Fatalf("old alpha log still exists or stat failed: %v", err)
	}
	for _, path := range []string{freshAlpha, oldBeta, keptDir} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}

func insertKanbanEventFixture(t *testing.T, store *Store, taskID, kind string, at time.Time) {
	t.Helper()
	if _, err := store.db.ExecContext(context.Background(), `INSERT INTO task_events(task_id, kind, payload, created_at) VALUES (?, ?, '', ?)`, taskID, kind, at.UTC().UnixMilli()); err != nil {
		t.Fatalf("insert fixture event %q: %v", kind, err)
	}
}

func writeKanbanLogFixture(t *testing.T, root, name string, modTime time.Time) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create log root: %v", err)
	}
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("log\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("age log fixture: %v", err)
	}
	return path
}
