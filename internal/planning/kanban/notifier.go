package kanban

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var defaultNotifyDeliveryKinds = []string{
	"completed",
	"blocked",
	"gave_up",
	"crashed",
	"timed_out",
	"worker_crashed",
	"worker_timed_out",
}

var terminalNotifyDeliveryKinds = map[string]struct{}{
	"completed":        {},
	"gave_up":          {},
	"crashed":          {},
	"timed_out":        {},
	"worker_crashed":   {},
	"worker_timed_out": {},
}

type NotifySender interface {
	SendKanbanNotification(context.Context, NotifyDelivery) error
}

type NotifyDelivery struct {
	Board        string
	Subscription NotifySubscription
	Task         Task
	Event        Event
	Message      string
}

type NotifyDeliveryFailure struct {
	Subscription NotifySubscription
	Event        Event
	Err          error
}

type NotifyRunResult struct {
	Delivered    []NotifyDelivery
	Unsubscribed []NotifySubscription
	Failed       []NotifyDeliveryFailure
}

type Notifier struct {
	Store  *Store
	Sender NotifySender
	Board  string
	Kinds  []string
}

func (n Notifier) RunOnce(ctx context.Context) (NotifyRunResult, error) {
	var result NotifyRunResult
	if n.Store == nil {
		return result, errors.New("kanban notifier store is required")
	}
	if n.Sender == nil {
		return result, errors.New("kanban notifier sender is required")
	}

	subscriptions, err := n.Store.ListNotifySubscriptions(ctx, "")
	if err != nil {
		return result, err
	}
	for _, sub := range subscriptions {
		input := notifySubscriptionInput(sub)
		cursor, events, err := n.Store.UnseenEventsForSubscription(ctx, sub.TaskID, input, n.deliveryKinds())
		if err != nil {
			return result, err
		}
		if len(events) == 0 {
			continue
		}

		task, taskTerminal, err := n.notifyTask(ctx, sub.TaskID)
		if err != nil {
			return result, err
		}
		deliveredAll := true
		lastKind := ""
		for _, event := range events {
			delivery := NotifyDelivery{
				Board:        notifyBoardName(n.Board),
				Subscription: sub,
				Task:         task,
				Event:        event,
				Message:      formatNotifyMessage(notifyBoardName(n.Board), sub, task, event),
			}
			if err := n.Sender.SendKanbanNotification(ctx, delivery); err != nil {
				result.Failed = append(result.Failed, NotifyDeliveryFailure{
					Subscription: sub,
					Event:        event,
					Err:          err,
				})
				deliveredAll = false
				break
			}
			result.Delivered = append(result.Delivered, delivery)
			lastKind = event.Kind
		}
		if !deliveredAll {
			continue
		}
		if cursor > sub.LastEventID {
			if err := n.Store.AdvanceNotifyCursor(ctx, sub.TaskID, input, cursor); err != nil {
				return result, err
			}
		}
		if taskTerminal || isTerminalNotifyDeliveryKind(lastKind) {
			removed, err := n.Store.RemoveNotifySubscription(ctx, sub.TaskID, input)
			if err != nil {
				return result, err
			}
			if removed {
				result.Unsubscribed = append(result.Unsubscribed, sub)
			}
		}
	}
	return result, nil
}

func (n Notifier) deliveryKinds() []string {
	if len(n.Kinds) == 0 {
		return append([]string(nil), defaultNotifyDeliveryKinds...)
	}
	return append([]string(nil), n.Kinds...)
}

func (n Notifier) notifyTask(ctx context.Context, taskID string) (Task, bool, error) {
	task, err := n.Store.GetTask(ctx, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return Task{ID: taskID, Title: taskID}, false, nil
		}
		return Task{}, false, err
	}
	return task, task.Status == StatusDone || task.Status == StatusArchived, nil
}

func notifySubscriptionInput(sub NotifySubscription) NotifySubscriptionInput {
	return NotifySubscriptionInput{
		Platform: sub.Platform,
		ChatID:   sub.ChatID,
		ThreadID: sub.ThreadID,
		UserID:   sub.UserID,
	}
}

func notifyBoardName(board string) string {
	board = strings.TrimSpace(board)
	if board == "" {
		return "default"
	}
	return board
}

func isTerminalNotifyDeliveryKind(kind string) bool {
	_, ok := terminalNotifyDeliveryKinds[strings.TrimSpace(kind)]
	return ok
}

func formatNotifyMessage(board string, sub NotifySubscription, task Task, event Event) string {
	title := firstLine(task.Title, 120)
	if title == "" {
		title = sub.TaskID
	}
	payload := firstLine(event.Payload, 200)
	assignee := strings.TrimSpace(task.Assignee)

	var b strings.Builder
	fmt.Fprintf(&b, "Kanban %s %s - %s", sub.TaskID, strings.TrimSpace(event.Kind), title)
	if assignee != "" {
		fmt.Fprintf(&b, " (@%s)", assignee)
	}
	if payload != "" {
		fmt.Fprintf(&b, ": %s", payload)
	}
	fmt.Fprintf(&b, " [board=%s]", notifyBoardName(board))
	return b.String()
}

func firstLine(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.IndexAny(value, "\r\n"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if limit > 0 && len(value) > limit {
		return value[:limit]
	}
	return value
}
