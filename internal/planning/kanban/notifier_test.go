package kanban

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestKanbanNotifierBlockedEventDoesNotUnsubscribe(t *testing.T) {
	ctx := context.Background()
	store := openNotifierTestStore(t)
	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "review docs", Assignee: "worker-a"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1", ThreadID: "thread-1"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}

	if err := store.BlockTask(ctx, task.ID, BlockTaskInput{Reason: "first block"}); err != nil {
		t.Fatalf("BlockTask(first) error = %v", err)
	}
	sender := &recordingKanbanNotifySender{}
	notifier := Notifier{Store: store, Sender: sender, Board: "default"}
	result, err := notifier.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce(first) error = %v", err)
	}
	if len(result.Delivered) != 1 {
		t.Fatalf("first Delivered = %+v, want one delivery", result.Delivered)
	}
	if !strings.Contains(sender.messages()[0], "blocked") || !strings.Contains(sender.messages()[0], "first block") {
		t.Fatalf("first message = %q, want blocked reason", sender.messages()[0])
	}

	if err := store.UnblockTask(ctx, task.ID); err != nil {
		t.Fatalf("UnblockTask() error = %v", err)
	}
	if err := store.BlockTask(ctx, task.ID, BlockTaskInput{Reason: "second block"}); err != nil {
		t.Fatalf("BlockTask(second) error = %v", err)
	}
	result, err = notifier.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce(second) error = %v", err)
	}
	if len(result.Delivered) != 1 {
		t.Fatalf("second Delivered = %+v, want one delivery", result.Delivered)
	}
	messages := sender.messages()
	if len(messages) != 2 {
		t.Fatalf("messages = %+v, want two blocked notifications", messages)
	}
	if !strings.Contains(messages[1], "second block") {
		t.Fatalf("second message = %q, want second block reason", messages[1])
	}
	subs, err := store.ListNotifySubscriptions(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListNotifySubscriptions() error = %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("subscriptions = %+v, want blocked notification subscription retained", subs)
	}
	if subs[0].LastEventID == 0 {
		t.Fatalf("LastEventID = 0, want cursor advanced after deliveries")
	}
}

func TestKanbanNotifierCompletedEventUnsubscribesAfterDelivery(t *testing.T) {
	ctx := context.Background()
	store := openNotifierTestStore(t)
	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "finish docs", Assignee: "worker-a"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}
	if err := store.CompleteTask(ctx, task.ID, CompleteTaskInput{Result: "done summary"}); err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}

	sender := &recordingKanbanNotifySender{}
	result, err := (Notifier{Store: store, Sender: sender, Board: "default"}).RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(result.Delivered) != 1 || len(result.Unsubscribed) != 1 {
		t.Fatalf("result = %+v, want one delivery and one unsubscribe", result)
	}
	if got := sender.messages(); len(got) != 1 || !strings.Contains(got[0], "completed") || !strings.Contains(got[0], "done summary") {
		t.Fatalf("messages = %+v, want completed summary", got)
	}
	subs, err := store.ListNotifySubscriptions(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListNotifySubscriptions() error = %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("subscriptions = %+v, want completed event to unsubscribe", subs)
	}
}

func TestKanbanNotifierAbnormalEventsUnsubscribe(t *testing.T) {
	for _, kind := range []string{"gave_up", "worker_crashed", "worker_timed_out"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			store := openNotifierTestStore(t)
			task, err := store.CreateTask(ctx, CreateTaskInput{Title: "unstable worker", Assignee: "worker-a"})
			if err != nil {
				t.Fatalf("CreateTask() error = %v", err)
			}
			input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1"}
			if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
				t.Fatalf("AddNotifySubscription() error = %v", err)
			}
			insertKanbanEventFixture(t, store, task.ID, kind, store.now())

			sender := &recordingKanbanNotifySender{}
			result, err := (Notifier{Store: store, Sender: sender, Board: "default"}).RunOnce(ctx)
			if err != nil {
				t.Fatalf("RunOnce() error = %v", err)
			}
			if len(result.Delivered) != 1 || len(result.Unsubscribed) != 1 {
				t.Fatalf("result = %+v, want one delivery and one unsubscribe", result)
			}
			if got := sender.messages(); len(got) != 1 || !strings.Contains(got[0], kind) {
				t.Fatalf("messages = %+v, want event kind %q", got, kind)
			}
			subs, err := store.ListNotifySubscriptions(ctx, task.ID)
			if err != nil {
				t.Fatalf("ListNotifySubscriptions() error = %v", err)
			}
			if len(subs) != 0 {
				t.Fatalf("subscriptions = %+v, want abnormal event to unsubscribe", subs)
			}
		})
	}
}

func TestKanbanNotifierSendFailureDoesNotAdvanceCursorOrUnsubscribe(t *testing.T) {
	ctx := context.Background()
	store := openNotifierTestStore(t)
	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "retry delivery"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	input := NotifySubscriptionInput{Platform: "telegram", ChatID: "chat-1"}
	if _, err := store.AddNotifySubscription(ctx, task.ID, input); err != nil {
		t.Fatalf("AddNotifySubscription() error = %v", err)
	}
	if err := store.BlockTask(ctx, task.ID, BlockTaskInput{Reason: "retry me"}); err != nil {
		t.Fatalf("BlockTask() error = %v", err)
	}

	result, err := (Notifier{Store: store, Sender: &recordingKanbanNotifySender{err: errors.New("temporary send failure")}}).RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(result.Failed) != 1 || len(result.Delivered) != 0 {
		t.Fatalf("result = %+v, want one failed delivery and no delivered messages", result)
	}
	subs, err := store.ListNotifySubscriptions(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListNotifySubscriptions() error = %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("subscriptions = %+v, want subscription retained", subs)
	}
	if subs[0].LastEventID != 0 {
		t.Fatalf("LastEventID = %d, want cursor unchanged after send failure", subs[0].LastEventID)
	}
}

func openNotifierTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

type recordingKanbanNotifySender struct {
	deliveries []NotifyDelivery
	err        error
}

func (s *recordingKanbanNotifySender) SendKanbanNotification(_ context.Context, delivery NotifyDelivery) error {
	if s.err != nil {
		return s.err
	}
	s.deliveries = append(s.deliveries, delivery)
	return nil
}

func (s *recordingKanbanNotifySender) messages() []string {
	out := make([]string, 0, len(s.deliveries))
	for _, delivery := range s.deliveries {
		out = append(out, delivery.Message)
	}
	return out
}
