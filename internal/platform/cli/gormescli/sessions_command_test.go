package gormescli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
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

// TestSessionExport_AcceptsIDPrefix proves the export command resolves
// a session-id prefix the same way `session delete` does. Without
// this, operators have to copy-paste the full id even though sibling
// commands accept the unique prefix — a small but real ergonomic
// inconsistency. The two commands are otherwise symmetric (both
// destructive in their own way: delete removes, export reads).
func TestSessionExport_AcceptsIDPrefix(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "20260315_092437_c9a6ff", title: "target", role: "user", content: "body of target", ts: 100},
		{id: "20260315_092500_other", role: "user", content: "other body", ts: 200},
	})

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "export", "20260315_092437")
	if err != nil {
		t.Fatalf("session export <prefix>: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "body of target") {
		t.Fatalf("export stdout missing seeded message body:\n%s", stdout)
	}
	// Must not export the other session.
	if strings.Contains(stdout, "other body") {
		t.Fatalf("export resolved to wrong session:\n%s", stdout)
	}
}

// TestSessionExport_AmbiguousPrefixSurfacesError proves that when the
// prefix is ambiguous, the command reports the conflict (matching the
// `session delete` shape) rather than silently picking one match.
func TestSessionExport_AmbiguousPrefixSurfacesError(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-ambig-a", role: "user", content: "alpha", ts: 100},
		{id: "sess-ambig-b", role: "user", content: "beta", ts: 200},
	})

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "export", "sess-ambig")
	if err == nil {
		t.Fatalf("ambiguous prefix must error; stdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "ambig") && !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("err should explain ambiguity; got %q", err)
	}
}

// TestSessionListJSONEmitsStructuredCatalog proves
// `gormes session list --json` emits a parseable
// `{build, sessions: [{id, title, preview, source, last_active_at, message_count}, ...]}`
// document. Fleet automation that wants to inventory sessions across
// hosts (which sessions are stale, which lack titles, message-count
// distribution) needs a structured shape — scraping the four-column
// human row is fragile when titles or previews contain spaces.
func TestSessionListJSONEmitsStructuredCatalog(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-alpha", title: "Alpha Work", role: "user", content: "preview alpha", ts: 100},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200},
	})

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "list", "--json")
	if err != nil {
		t.Fatalf("session list --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Sessions []struct {
			ID           string `json:"id"`
			Title        string `json:"title"`
			Preview      string `json:"preview"`
			Source       string `json:"source"`
			MessageCount int    `json:"message_count"`
			LastActiveAt int64  `json:"last_active_at"`
		} `json:"sessions"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if len(got.Sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(got.Sessions))
	}
	// Verify ID is present for all entries — that's the resume handle.
	for i, s := range got.Sessions {
		if s.ID == "" {
			t.Fatalf("session[%d].ID empty:\n%s", i, stdout)
		}
	}
	// JSON mode must not interleave the human header row.
	if strings.Contains(stdout, "Title") && strings.Contains(stdout, "Preview") && strings.Contains(stdout, "Last Active") {
		t.Fatalf("--json must not emit the human header; got:\n%s", stdout)
	}
}

// TestSessionListJSONEmptyDirectoryEmitsEmptyArray proves the JSON
// surface stays parseable when no sessions exist — consumers see
// `{"sessions": []}`, not the free-form "No sessions found." message.
func TestSessionListJSONEmptyDirectoryEmitsEmptyArray(t *testing.T) {
	seedSessionsCommandDB(t, nil)

	stdout, _, err := runSessionsCommand(t, nil, "session", "list", "--json")
	if err != nil {
		t.Fatalf("session list --json on empty: %v", err)
	}
	var got struct {
		Sessions []any `json:"sessions"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Sessions == nil {
		t.Fatalf("sessions must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("got %d sessions, want 0", len(got.Sessions))
	}
}

func TestSessionListAndExportDeduplicateMirroredChatRows(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-live", role: "user", content: "hello", ts: 100, turnKey: "turn-user"},
		{id: "sess-live", role: "user", content: "hello", ts: 100, chatID: "user"},
		{id: "sess-live", role: "assistant", content: "hi", ts: 101, turnKey: "turn-agent"},
		{id: "sess-live", role: "assistant", content: "hi", ts: 101, chatID: "gormes"},
	})
	writeSessionMirrorIndex(t, "sess-live", "telegram")

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "list", "--json")
	if err != nil {
		t.Fatalf("session list --json: %v\nstderr=%s", err, stderr)
	}
	var list struct {
		Sessions []struct {
			ID           string `json:"id"`
			Source       string `json:"source"`
			MessageCount int    `json:"message_count"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("list JSON: %v\nstdout=%s", err, stdout)
	}
	if len(list.Sessions) != 1 {
		t.Fatalf("got %d sessions, want 1: %s", len(list.Sessions), stdout)
	}
	if list.Sessions[0].MessageCount != 2 {
		t.Fatalf("message_count = %d, want deduped 2; stdout=%s", list.Sessions[0].MessageCount, stdout)
	}
	if list.Sessions[0].Source != "telegram" {
		t.Fatalf("source = %q, want telegram from mirrored channel row", list.Sessions[0].Source)
	}

	export, stderr, err := runSessionsCommand(t, nil, "session", "export", "sess-live")
	if err != nil {
		t.Fatalf("session export: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(export, "**Platform:** telegram") {
		t.Fatalf("export should prefer session mirror source telegram:\n%s", export)
	}
	if !strings.Contains(export, "**Messages:** 2") {
		t.Fatalf("export should report deduped message count 2:\n%s", export)
	}
	if strings.Contains(export, "## Turn 3") || strings.Count(export, "**User:** hello") != 1 || strings.Count(export, "**Agent:** hi") != 1 {
		t.Fatalf("export contains duplicate mirrored turns:\n%s", export)
	}
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

	got, err := ResolveContinueSessionFlag("last")
	if err != nil {
		t.Fatalf("resolveContinueSessionFlag: %v", err)
	}
	if got != "sess-old-start" {
		t.Fatalf("continue resolved %q, want sess-old-start", got)
	}
}

func TestCoalesceSessionArgsMultiWordNames(t *testing.T) {
	got := CoalesceSessionNameArgs([]string{"-c", "my", "project", "sessions", "list"})
	want := []string{"-c", "my project", "sessions", "list"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("coalesced args = %#v, want %#v", got, want)
	}

	got = CoalesceSessionNameArgs([]string{"--resume", "deep", "work", "--offline"})
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
	chatID  string
	turnKey string
}

// TestSessionDelete_JSONEmitsStructuredOutcome proves
// `gormes session delete <id> --yes --json` returns
// `{build, action, requested_id, resolved_id, deleted}` so fleet
// automation tearing down sessions can audit which sessions actually
// vanished per machine. `requested_id` echoes the user input
// (possibly a prefix); `resolved_id` is the full id that was deleted.
func TestSessionDelete_JSONEmitsStructuredOutcome(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "20260315_092437_c9a6ff", role: "user", content: "target", ts: 100},
	})

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "delete", "20260315_092437", "--yes", "--json")
	if err != nil {
		t.Fatalf("session delete --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action      string `json:"action"`
		RequestedID string `json:"requested_id"`
		ResolvedID  string `json:"resolved_id"`
		Deleted     bool   `json:"deleted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("session delete --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "deleted" {
		t.Errorf("action = %q, want %q", got.Action, "deleted")
	}
	if got.RequestedID != "20260315_092437" {
		t.Errorf("requested_id = %q, want %q", got.RequestedID, "20260315_092437")
	}
	if got.ResolvedID != "20260315_092437_c9a6ff" {
		t.Errorf("resolved_id = %q, want full id %q", got.ResolvedID, "20260315_092437_c9a6ff")
	}
	if !got.Deleted {
		t.Errorf("deleted must be true on successful delete")
	}
	assertSessionCommandTurnCount(t, "20260315_092437_c9a6ff", 0)
}

// TestSessionDelete_JSONNotFoundEmitsParseable proves the
// not-found path also returns a JSON document (with deleted=false,
// action="not_found") rather than fall back to prose. Fleet scripts
// iterating across hosts need a stable parseable signal whether the
// session existed or not.
func TestSessionDelete_JSONNotFoundEmitsParseable(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "exists", role: "user", content: "x", ts: 100},
	})

	stdout, _, err := runSessionsCommand(t, nil, "session", "delete", "does-not-exist", "--yes", "--json")
	if err != nil {
		t.Fatalf("session delete --json (missing): %v\nstdout=%s", err, stdout)
	}

	var got struct {
		Action      string `json:"action"`
		RequestedID string `json:"requested_id"`
		Deleted     bool   `json:"deleted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("session delete --json (missing) must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Action != "not_found" {
		t.Errorf("action = %q, want %q", got.Action, "not_found")
	}
	if got.Deleted {
		t.Errorf("deleted must be false when session not found")
	}
}

// TestSessionPrune_JSONEmitsStructuredOutcome proves
// `gormes session prune --older-than N --yes --json` returns a
// parseable `{build, action, older_than_days, source, pruned}`
// document so fleet automation running scheduled session GC can
// audit how many sessions vanished per machine without scraping
// "Pruned N session(s)." prose.
func TestSessionPrune_JSONEmitsStructuredOutcome(t *testing.T) {
	// Seed an old session (ts=1) and a fresh one. Setting --older-than=1
	// means anything older than 1 day from now is pruned — both seeds
	// have ts=1 epoch so they are both well past the cutoff.
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "old-session-1", role: "user", content: "old1", ts: 1},
		{id: "old-session-2", role: "user", content: "old2", ts: 1},
	})

	stdout, stderr, err := runSessionsCommand(t, nil, "session", "prune", "--older-than=1", "--yes", "--json")
	if err != nil {
		t.Fatalf("session prune --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action        string `json:"action"`
		OlderThanDays int    `json:"older_than_days"`
		Source        string `json:"source"`
		Pruned        int    `json:"pruned"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("session prune --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "pruned" {
		t.Errorf("action = %q, want %q", got.Action, "pruned")
	}
	if got.OlderThanDays != 1 {
		t.Errorf("older_than_days = %d, want 1", got.OlderThanDays)
	}
	if got.Pruned != 2 {
		t.Errorf("pruned = %d, want 2 (both old sessions)", got.Pruned)
	}
	assertSessionCommandTurnCount(t, "old-session-1", 0)
	assertSessionCommandTurnCount(t, "old-session-2", 0)
}

func writeSessionMirrorIndex(t *testing.T, sessionID, source string) {
	t.Helper()
	path := config.SessionIndexMirrorPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create session mirror dir: %v", err)
	}
	body := "# Auto-generated session index\nsessions:\n  " + source + ":42: " + sessionID + "\nlineage:\n  " + sessionID + ":\n    lineage_kind: primary\n    lineage_status: ok\n    source: " + source + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write session mirror: %v", err)
	}
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
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json, turn_key) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
			seed.id, seed.role, seed.content, seed.ts, seed.chatID, meta, seed.turnKey,
		); err != nil {
			t.Fatalf("seed session %s: %v", seed.id, err)
		}
	}
}

func runSessionsCommand(t *testing.T, in *strings.Reader, args ...string) (string, string, error) {
	t.Helper()
	// Each newSessionRootCommandForTest() builds a fresh session command tree via
	// newSessionCommandForTest(), so flag state is naturally isolated — no
	// explicit per-flag reset needed.
	cmd := newSessionRootCommandForTest()
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
// each newSessionCommandForTest() returns a fresh tree. No more
// resetSessionsCommandFlagState workaround needed.
func TestSessionCommand_ConstructorReturnsIndependentInstances(t *testing.T) {
	a := newSessionCommandForTest()
	b := newSessionCommandForTest()
	if a == b {
		t.Fatal("newSessionCommand must return distinct instances")
	}
	const want = 8
	if len(a.Commands()) != want || len(b.Commands()) != want {
		t.Fatalf("session tree must have %d subcommands; got len(a)=%d len(b)=%d", want, len(a.Commands()), len(b.Commands()))
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
