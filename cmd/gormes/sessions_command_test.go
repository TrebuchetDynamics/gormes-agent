package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestSessionsDeletePrefixResolution(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "20260315_092437_c9a6ff", role: "user", content: "target", ts: 100},
		{id: "20260315_092500_other", role: "user", content: "other", ts: 200},
	})

	stdout, stderr, err := runSessionsCommand(t, nil, "sessions", "delete", "20260315_092437_c9a6", "--yes")
	if err != nil {
		t.Fatalf("sessions delete: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted session '20260315_092437_c9a6ff'.") {
		t.Fatalf("stdout = %q, want deleted full id", stdout)
	}
	assertSessionCommandTurnCount(t, "20260315_092437_c9a6ff", 0)
	assertSessionCommandTurnCount(t, "20260315_092500_other", 1)
}

func TestSessionsDeleteAndPruneEOFConfirmCancel(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-delete", role: "user", content: "target", ts: 100},
		{id: "sess-prune", role: "user", content: "old", ts: 1},
	})

	stdout, _, err := runSessionsCommand(t, strings.NewReader(""), "sessions", "delete", "sess-delete")
	if err != nil {
		t.Fatalf("sessions delete EOF: %v", err)
	}
	if !strings.Contains(stdout, "Cancelled.") {
		t.Fatalf("delete stdout = %q, want Cancelled", stdout)
	}
	assertSessionCommandTurnCount(t, "sess-delete", 1)

	stdout, _, err = runSessionsCommand(t, strings.NewReader(""), "sessions", "prune", "--older-than=1")
	if err != nil {
		t.Fatalf("sessions prune EOF: %v", err)
	}
	if !strings.Contains(stdout, "Cancelled.") {
		t.Fatalf("prune stdout = %q, want Cancelled", stdout)
	}
	assertSessionCommandTurnCount(t, "sess-prune", 1)
}

func TestSessionsBrowseFallback(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-alpha", title: "Alpha Work", role: "user", content: "preview alpha", ts: 100},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200},
	})

	stdout, _, err := runSessionsCommand(t, strings.NewReader("x\n1\n"), "sessions", "browse", "--no-curses")
	if err != nil {
		t.Fatalf("sessions browse: %v", err)
	}
	if !strings.Contains(stdout, "Invalid input. Enter a number or q to cancel.") {
		t.Fatalf("stdout = %q, want invalid retry", stdout)
	}
	if !strings.Contains(stdout, "Beta Work") || !strings.Contains(stdout, "preview beta") {
		t.Fatalf("stdout = %q, want title-over-preview display", stdout)
	}
	if !strings.Contains(stdout, "Resuming session: sess-beta") {
		t.Fatalf("stdout = %q, want selected session", stdout)
	}
}

func TestSessionsContinueResolvesMostRecentlyActive(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-old-start", role: "user", content: "old start", ts: 100},
		{id: "sess-new-start", role: "user", content: "new start", ts: 200},
		{id: "sess-old-start", role: "assistant", content: "recent activity", ts: 300},
	})

	got, err := resolveContinueSessionFlag("last")
	if err != nil {
		t.Fatalf("resolveContinueSessionFlag: %v", err)
	}
	if got != "sess-old-start" {
		t.Fatalf("continue resolved %q, want sess-old-start", got)
	}
}

func TestCoalesceSessionArgsMultiWordNames(t *testing.T) {
	got := coalesceSessionNameArgs([]string{"-c", "my", "project", "sessions", "list"})
	want := []string{"-c", "my project", "sessions", "list"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("coalesced args = %#v, want %#v", got, want)
	}

	got = coalesceSessionNameArgs([]string{"--resume", "deep", "work", "--offline"})
	want = []string{"--resume", "deep work", "--offline"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("coalesced resume args = %#v, want %#v", got, want)
	}
}

type sessionCommandSeed struct {
	id      string
	title   string
	role    string
	content string
	ts      int64
}

func seedSessionsCommandDB(t *testing.T, seeds []sessionCommandSeed) {
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

	for _, seed := range seeds {
		meta := ""
		if seed.title != "" {
			meta = `{"title":"` + seed.title + `"}`
		}
		if _, err := store.DB().Exec(
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json) VALUES (?, ?, ?, ?, '', NULLIF(?, ''))`,
			seed.id, seed.role, seed.content, seed.ts, meta,
		); err != nil {
			t.Fatalf("seed session %s: %v", seed.id, err)
		}
	}
}

func runSessionsCommand(t *testing.T, in *strings.Reader, args ...string) (string, string, error) {
	t.Helper()
	// Each newRootCommand() builds a fresh session command tree via
	// newSessionCommand(), so flag state is naturally isolated — no
	// explicit per-flag reset needed.
	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if in != nil {
		cmd.SetIn(in)
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// TestSessionCommand_ConstructorReturnsIndependentInstances proves
// each newSessionCommand() returns a fresh tree. No more
// resetSessionsCommandFlagState workaround needed.
func TestSessionCommand_ConstructorReturnsIndependentInstances(t *testing.T) {
	a := newSessionCommand()
	b := newSessionCommand()
	if a == b {
		t.Fatal("newSessionCommand must return distinct instances")
	}
	if len(a.Commands()) != 5 || len(b.Commands()) != 5 {
		t.Fatalf("session tree must have 5 subcommands; got len(a)=%d len(b)=%d", len(a.Commands()), len(b.Commands()))
	}
	for i := range a.Commands() {
		if a.Commands()[i] == b.Commands()[i] {
			t.Fatalf("subcommand[%d] %q shared between constructor calls", i, a.Commands()[i].Use)
		}
	}
}

func assertSessionCommandTurnCount(t *testing.T, sessionID string, want int) {
	t.Helper()
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())
	var got int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM turns WHERE session_id = ?`, sessionID).Scan(&got); err != nil {
		t.Fatalf("count turns for %s: %v", sessionID, err)
	}
	if got != want {
		t.Fatalf("turn count for %s = %d, want %d", sessionID, got, want)
	}
}
