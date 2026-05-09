package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

// TestSessionExportCommand_JSONEmitsStructuredTranscript proves
// `gormes session export <id> --json` emits a parseable
// `{build, session_id, format, content}` envelope so fleet automation
// aggregating exported session transcripts across machines can ingest
// each export with binary attribution and stable shape, instead of
// concatenating raw markdown blobs. Build provenance leads — same
// convention as the rest of the `--json` arc.
func TestSessionExportCommand_JSONEmitsStructuredTranscript(t *testing.T) {
	seedTranscriptDB(t, "sess-cli-json", "discord:chan-9")

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"session", "export", "sess-cli-json", "--format=markdown", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("session export --json: %v\nstderr=%s", err, stderr.String())
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		SessionID string `json:"session_id"`
		Format    string `json:"format"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.SessionID != "sess-cli-json" {
		t.Errorf("session_id = %q, want sess-cli-json", got.SessionID)
	}
	if got.Format != "markdown" {
		t.Errorf("format = %q, want markdown", got.Format)
	}
	if !strings.Contains(got.Content, "# Session: sess-cli-json") {
		t.Errorf("content missing session heading:\n%s", got.Content)
	}
}

func TestSessionExportCommand_Markdown(t *testing.T) {
	seedTranscriptDB(t, "sess-cli", "discord:chan-7")

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"session", "export", "sess-cli", "--format=markdown"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "# Session: sess-cli") {
		t.Fatalf("stdout = %q, want session heading", out)
	}
	if !strings.Contains(out, "**Tool Calls:**") {
		t.Fatalf("stdout = %q, want tool call section", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestSessionExportCommand_MissingSession(t *testing.T) {
	seedTranscriptDB(t, "other-session", "telegram:11")

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"session", "export", "missing", "--format=markdown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want missing-session error")
	}
	if !strings.Contains(err.Error(), `session "missing" not found`) {
		t.Fatalf("error = %v, want missing-session message", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func seedTranscriptDB(t *testing.T, sessionID, chatID string) {
	t.Helper()

	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(dataHome, "gormes"))
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	meta := `{"tool_calls":[{"id":"call_1","name":"list_dir","arguments":{"path":"."}}]}`
	if _, err := store.DB().Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json)
		 VALUES (?, 'user', 'hello', ?, ?, NULL),
		        (?, 'assistant', 'working on it', ?, ?, ?)`,
		sessionID, time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC).Unix(), chatID,
		sessionID, time.Date(2026, 4, 21, 9, 0, 3, 0, time.UTC).Unix(), chatID, meta,
	); err != nil {
		t.Fatalf("seed turns: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(config.MemoryDBPath()), 0o755); err != nil {
		t.Fatalf("MkdirAll(data dir): %v", err)
	}
}
