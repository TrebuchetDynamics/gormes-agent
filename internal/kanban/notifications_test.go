package kanban

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKanbanNotificationOnComplete(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "test task"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}
	if err := store.CompleteTask(ctx, task.ID, CompleteTaskInput{Result: "done"}); err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}

	sender := &recordingKanbanNotifySender{}
	notifier := Notifier{Store: store, Sender: sender, Board: "default"}
	result, err := notifier.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(result.Delivered) != 1 {
		t.Fatalf("Delivered = %+v, want one delivery for completed task", result.Delivered)
	}
	msg := sender.messages()[0]
	if !strings.Contains(msg, "completed") {
		t.Fatalf("message = %q, want completed event kind", msg)
	}
}

func TestKanbanNotificationOnFailure(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "failing task"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}

	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	insertKanbanEventFixture(t, store, task.ID, "crashed", now)

	sender := &recordingKanbanNotifySender{}
	notifier := Notifier{Store: store, Sender: sender, Board: "default"}
	result, err := notifier.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(result.Delivered) != 1 {
		t.Fatalf("Delivered = %+v, want one delivery for failed task", result.Delivered)
	}
	msg := sender.messages()[0]
	if !strings.Contains(msg, "crashed") {
		t.Fatalf("message = %q, want crashed event kind", msg)
	}
}

func TestKanbanNotificationThrottle(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "throttle test task"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}

	fakeNow := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	// Non-terminal events so the subscription survives across runs.
	insertKanbanEventFixture(t, store, task.ID, "commented", fakeNow)
	insertKanbanEventFixture(t, store, task.ID, "assigned", fakeNow.Add(time.Second))
	insertKanbanEventFixture(t, store, task.ID, "blocked", fakeNow.Add(2*time.Second))

	recorder := &recordingKanbanNotifySender{}
	throttled := NewThrottledNotifySenderWithClock(recorder, 1*time.Minute, func() time.Time { return fakeNow })
	// Explicit Kinds so the Notifier sees all test event kinds.
	notifier := Notifier{Store: store, Sender: throttled, Board: "default", Kinds: []string{"commented", "assigned", "blocked", "unblocked"}}

	// First run: 3 events, throttle allows first and drops subsequent two.
	if _, err := notifier.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce(first) error = %v", err)
	}
	if got := len(recorder.messages()); got != 1 {
		t.Fatalf("after first run recorder has %d messages, want 1 (throttle allows first only)", got)
	}

	// Advance clock past the throttle window; subscription still exists for non-terminal events.
	fakeNow = fakeNow.Add(2 * time.Minute)
	throttled.now = func() time.Time { return fakeNow }
	insertKanbanEventFixture(t, store, task.ID, "unblocked", fakeNow)

	if _, err := notifier.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce(second) error = %v", err)
	}
	if got := len(recorder.messages()); got != 2 {
		t.Fatalf("after second run recorder has %d messages, want 2 (throttle allows after window)", got)
	}
}
