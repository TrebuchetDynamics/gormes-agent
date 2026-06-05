package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestGonchoMemoryCommand_SearchDisplaysResults(t *testing.T) {
	setupGonchoDoctorEnv(t)
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "test"}, nil)

	ctx := context.Background()
	_, err = svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "operator",
		Conclusion: "test decision about SQLite migration",
	})
	if err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	cmd := newGonchoMemorySearchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"SQLite", "--peer", "operator"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected search output, got empty string")
	}
}

func TestGonchoMemoryCommand_SearchJSON(t *testing.T) {
	setupGonchoDoctorEnv(t)
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "test"}, nil)

	ctx := context.Background()
	_, err = svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "operator",
		Conclusion: "test JSON output decision",
	})
	if err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	cmd := newGonchoMemorySearchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"JSON", "--peer", "operator", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var result goncho.SearchResultSet
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON output: %v\noutput: %s", err, buf.String())
	}
}

func TestGonchoContinueCommand_ListsRecentSessions(t *testing.T) {
	setupGonchoDoctorEnv(t)
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "test"}, nil)

	ctx := context.Background()
	_, err = svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "operator",
		Conclusion: "test session summary",
	})
	if err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	cmd := newGonchoContinueCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected output, got empty string")
	}
}
