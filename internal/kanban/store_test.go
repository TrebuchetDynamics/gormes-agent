package kanban

import (
	"context"
	"path/filepath"
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
