package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
)

func TestKanbanNotifySubscribeListUnsubscribe(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	task := runKanbanJSONTask(t, "create", "Notify CLI task", "--json")

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"kanban", "notify-subscribe", task.ID,
		"--platform", "telegram",
		"--chat-id", "chat-42",
		"--thread-id", "thread-7",
		"--user-id", "user-9",
	)
	if err != nil {
		t.Fatalf("kanban notify-subscribe: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Subscribed telegram:chat-42:thread-7 to "+task.ID) {
		t.Fatalf("notify-subscribe stdout = %q, want Hermes-shaped subscription text", stdout)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "notify-list", task.ID, "--json")
	if err != nil {
		t.Fatalf("kanban notify-list --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var list struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Subscriptions []kanban.NotifySubscription `json:"subscriptions"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &list); jsonErr != nil {
		t.Fatalf("notify-list JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if list.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", list.Build.Version, Version)
	}
	if len(list.Subscriptions) != 1 {
		t.Fatalf("subscriptions = %+v, want one", list.Subscriptions)
	}
	sub := list.Subscriptions[0]
	if sub.TaskID != task.ID || sub.Platform != "telegram" || sub.ChatID != "chat-42" || sub.ThreadID != "thread-7" || sub.UserID != "user-9" || sub.LastEventID != 0 {
		t.Fatalf("subscription = %+v, want task/platform/chat/thread/user/cursor evidence", sub)
	}

	stdout, stderr, err = executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"kanban", "notify-unsubscribe", task.ID,
		"--platform", "telegram",
		"--chat-id", "chat-42",
		"--thread-id", "thread-7",
	)
	if err != nil {
		t.Fatalf("kanban notify-unsubscribe: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Unsubscribed from "+task.ID) {
		t.Fatalf("notify-unsubscribe stdout = %q, want unsubscribe text", stdout)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "notify-list", task.ID)
	if err != nil {
		t.Fatalf("kanban notify-list after unsubscribe: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "(no subscriptions)") {
		t.Fatalf("notify-list stdout = %q, want empty subscription text", stdout)
	}

	stdout, stderr, err = executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"kanban", "notify-unsubscribe", task.ID,
		"--platform", "telegram",
		"--chat-id", "chat-42",
		"--thread-id", "thread-7",
	)
	if err == nil {
		t.Fatalf("kanban notify-unsubscribe missing subscription succeeded\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "no such subscription") {
		t.Fatalf("missing subscription output = %q, want bounded not-found evidence", stdout+stderr+err.Error())
	}
}

func TestKanbanNotifyListJSONEmitsEmptyArray(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "notify-list", "--json")
	if err != nil {
		t.Fatalf("kanban notify-list --json empty: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var list struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Subscriptions []kanban.NotifySubscription `json:"subscriptions"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &list); jsonErr != nil {
		t.Fatalf("notify-list empty JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if list.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", list.Build.Version, Version)
	}
	if list.Subscriptions == nil || len(list.Subscriptions) != 0 {
		t.Fatalf("subscriptions = %#v, want non-nil empty array", list.Subscriptions)
	}
}
