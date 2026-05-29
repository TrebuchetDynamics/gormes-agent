package kanban

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestNotifySubscriptionAddListRemove(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 5, 9, 12, 30, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "Notify task"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{
		Platform: "telegram",
		ChatID:   "chat-42",
		ThreadID: "thread-7",
		UserID:   "user-9",
	}

	sub, err := store.AddNotifySubscription(ctx, task.ID, input)
	if err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}
	if sub.TaskID != task.ID || sub.Platform != input.Platform || sub.ChatID != input.ChatID || sub.ThreadID != input.ThreadID || sub.UserID != input.UserID {
		t.Fatalf("subscription = %+v, want task/platform/chat/thread/user fields", sub)
	}
	if sub.LastEventID != 0 {
		t.Fatalf("LastEventID = %d, want 0", sub.LastEventID)
	}
	if !sub.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %s, want %s", sub.CreatedAt, now)
	}

	duplicate, err := store.AddNotifySubscription(ctx, task.ID, input)
	if err != nil {
		t.Fatalf("AddNotifySubscription(duplicate) error = %v", err)
	}
	if duplicate.LastEventID != 0 || !duplicate.CreatedAt.Equal(now) {
		t.Fatalf("duplicate subscription = %+v, want original cursor/time", duplicate)
	}

	filtered, err := store.ListNotifySubscriptions(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListNotifySubscriptions(task) error = %v", err)
	}
	all, err := store.ListNotifySubscriptions(ctx, "")
	if err != nil {
		t.Fatalf("ListNotifySubscriptions(all) error = %v", err)
	}
	if len(filtered) != 1 || len(all) != 1 {
		t.Fatalf("subscriptions filtered=%+v all=%+v, want one subscription in both views", filtered, all)
	}

	removed, err := store.RemoveNotifySubscription(ctx, task.ID, NotifySubscriptionInput{
		Platform: input.Platform,
		ChatID:   input.ChatID,
		ThreadID: input.ThreadID,
	})
	if err != nil {
		t.Fatalf("RemoveNotifySubscription() error = %v", err)
	}
	if !removed {
		t.Fatal("RemoveNotifySubscription() removed = false, want true")
	}
	removed, err = store.RemoveNotifySubscription(ctx, task.ID, input)
	if err != nil {
		t.Fatalf("RemoveNotifySubscription(missing) error = %v", err)
	}
	if removed {
		t.Fatal("RemoveNotifySubscription(missing) removed = true, want false")
	}
}

func TestNotifySubscriptionEventsAndCursor(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "Cursor task"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `DELETE FROM task_events WHERE task_id = ?`, task.ID); err != nil {
		t.Fatalf("reset task_events fixture: %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-42", ThreadID: "thread-7"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}

	base := time.Date(2026, 5, 9, 13, 0, 0, 0, time.UTC)
	insertKanbanEventFixture(t, store, task.ID, "commented", base)
	insertKanbanEventFixture(t, store, task.ID, "blocked", base.Add(time.Second))
	insertKanbanEventFixture(t, store, task.ID, "completed", base.Add(2*time.Second))

	cursor, events, err := store.UnseenEventsForSubscription(ctx, task.ID, input, nil)
	if err != nil {
		t.Fatalf("UnseenEventsForSubscription() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("unseen events = %+v, want 3", events)
	}
	if cursor != events[2].ID {
		t.Fatalf("cursor = %d, want latest event id %d", cursor, events[2].ID)
	}

	filterCursor, filtered, err := store.UnseenEventsForSubscription(ctx, task.ID, input, []string{"blocked"})
	if err != nil {
		t.Fatalf("UnseenEventsForSubscription(blocked) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].Kind != "blocked" || filterCursor != filtered[0].ID {
		t.Fatalf("filtered events = %+v cursor=%d, want one blocked event cursor", filtered, filterCursor)
	}

	if err := store.AdvanceNotifyCursor(ctx, task.ID, input, cursor); err != nil {
		t.Fatalf("AdvanceNotifyCursor() error = %v", err)
	}
	afterCursor, after, err := store.UnseenEventsForSubscription(ctx, task.ID, input, nil)
	if err != nil {
		t.Fatalf("UnseenEventsForSubscription(after cursor) error = %v", err)
	}
	if len(after) != 0 || afterCursor != cursor {
		t.Fatalf("after cursor events = %+v cursor=%d, want no events and cursor %d", after, afterCursor, cursor)
	}
}
