package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
)

func TestBuildDefaultRegistryKanbanHiddenByDefault(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_TASK", "")

	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	if _, ok := reg.Get("kanban_show"); ok {
		t.Fatal("kanban_show registered without Kanban worker context")
	}
	if _, ok := reg.Get("kanban_list"); ok {
		t.Fatal("kanban_list registered without Kanban worker or orchestrator context")
	}
}

func TestBuildDefaultRegistryKanbanVisibleWithWorkerEnv(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "kanban.db")
	store, err := kanban.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	task, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "worker task", Assignee: "coder"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, claimed, err := store.ClaimTask(ctx, task.ID, kanban.ClaimTaskInput{Worker: "kanban-dispatcher", TTL: time.Minute}); err != nil || !claimed {
		t.Fatalf("ClaimTask: claimed=%v err=%v", claimed, err)
	}

	t.Setenv("GORMES_KANBAN_DB", dbPath)
	t.Setenv("GORMES_KANBAN_TASK", task.ID)
	t.Setenv("GORMES_PROFILE", "coder")

	reg := buildDefaultRegistry(ctx, config.Config{}, nil, "")
	for _, name := range []string{"kanban_show", "kanban_complete", "kanban_block", "kanban_heartbeat", "kanban_comment", "kanban_create", "kanban_link"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered with Kanban worker env", name)
		}
	}
	for _, name := range []string{"kanban_list", "kanban_unblock"} {
		if _, ok := reg.Get(name); ok {
			t.Fatalf("%s registered for scoped Kanban worker env", name)
		}
	}

	show, ok := reg.Get("kanban_show")
	if !ok {
		t.Fatal("kanban_show missing")
	}
	raw, err := show.Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("kanban_show Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal kanban_show output %s: %v", raw, err)
	}
	if out["worker_context"] == "" {
		t.Fatalf("kanban_show output missing worker_context: %v", out)
	}
}

func TestBuildDefaultRegistryKanbanOrchestratorToolsetIncludesBoardRouting(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("GORMES_KANBAN_DB", filepath.Join(t.TempDir(), "kanban.db"))
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_TASK", "")
	t.Setenv("GORMES_TOOLSETS", "kanban")

	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	for _, name := range []string{"kanban_show", "kanban_list", "kanban_complete", "kanban_block", "kanban_heartbeat", "kanban_comment", "kanban_create", "kanban_link", "kanban_unblock"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered with orchestrator kanban toolset", name)
		}
	}
}
