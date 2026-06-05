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

func TestMemoryStatusCommand_PrintsExtractorSummary(t *testing.T) {
	seedMemoryStatusDB(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Extractor status",
		"worker_health: degraded",
		"queue_depth: 1",
		"dead_letters: 2",
		"dead_letter_summary:",
		"error=\"malformed JSON\" count=1",
		"error=\"upstream timeout\" count=1",
		"session_id=sess-3",
		"error=\"upstream timeout\"",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want substring %q", out, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestMemoryStatusCommand_MissingDatabase pins the empty-state UX:
// on a fresh install the goncho memory.db doesn't exist yet (it's
// created lazily on the first turn write), so a read-only inventory
// command must report "queue empty / no DLQ entries / 0 jobs" rather
// than raise a "memory database not found" error. Same shape as the
// `session list` empty-state fence.
//
// Operator pain this prevents: `gormes memory status` is the natural
// command an SRE runs to confirm the worker queue is healthy on a
// freshly-imaged host. Erroring out makes that healthy state look
// indistinguishable from a corrupt DB or permission error.
func TestMemoryStatusCommand_MissingDatabase(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(dataHome, "gormes"))

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() on fresh install must succeed; got %v\nstderr=%s", err, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Extractor status",
		"queue_depth: 0",
		"dead_letters: 0",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q on fresh-install zero state:\n%s", want, out)
		}
	}
}

// TestMemoryStatusCommand_EmptyDatabase_NoSchemaIsZeroState pins a
// second corner of the empty-state UX: an existing-but-zero-byte
// memory.db (observed in production after install/upgrade flows that
// touched the file path but didn't run schema migrations) must not
// surface a raw `sqlite3: SQL logic error: no such table: turns` to
// the operator. Same empty-state contract as the missing-file path.
func TestMemoryStatusCommand_EmptyDatabase_NoSchemaIsZeroState(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	gormesHome := filepath.Join(dataHome, "gormes")
	t.Setenv("GORMES_HOME", gormesHome)
	if err := os.MkdirAll(gormesHome, 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed a 0-byte memory.db — exists but no goncho schema applied.
	if err := os.WriteFile(filepath.Join(gormesHome, "memory.db"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("memory status on 0-byte memory.db must succeed; got %v\nstderr=%s", err, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"Extractor status", "queue_depth: 0", "dead_letters: 0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q on schema-less DB:\n%s", want, out)
		}
	}
	// The raw sqlite error must not leak through.
	combined := out + stderr.String()
	if strings.Contains(combined, "no such table") || strings.Contains(combined, "SQL logic error") {
		t.Fatalf("raw sqlite error leaked to operator: %q", combined)
	}
}

func TestMemoryStatusCommand_CorruptDatabaseSelfHealsToZeroState(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	gormesHome := filepath.Join(dataHome, "gormes")
	t.Setenv("GORMES_HOME", gormesHome)
	if err := os.MkdirAll(gormesHome, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(gormesHome, "memory.db")
	if err := os.WriteFile(dbPath, []byte("not sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("memory status --json must self-heal corrupt memory.db; got %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}

	var got struct {
		Extractor struct {
			QueueDepth      int `json:"queue_depth"`
			DeadLetterCount int `json:"dead_letter_count"`
		} `json:"extractor"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout.String())
	}
	if got.Extractor.QueueDepth != 0 || got.Extractor.DeadLetterCount != 0 {
		t.Fatalf("self-healed corrupt memory DB should report zero queue state, got %+v", got.Extractor)
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "file is not a database") || strings.Contains(combined, "SQL logic error") {
		t.Fatalf("raw sqlite corruption leaked to operator: %q", combined)
	}
	backups, err := filepath.Glob(dbPath + ".corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("corrupt memory.db must be preserved as one quarantine backup, got %v", backups)
	}
}

// TestMemoryStatusCommand_MissingDatabase_JSONEmitsZeroState keeps the
// JSON surface symmetric: SREs scraping
// `gormes memory status --json` from a freshly-imaged host should see
// `{"extractor": {"queue_depth": 0, ...}}`, not a non-zero exit.
func TestMemoryStatusCommand_MissingDatabase_JSONEmitsZeroState(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(dataHome, "gormes"))

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("memory status --json on fresh install: %v\nstderr=%s", err, stderr.String())
	}

	var got struct {
		Extractor struct {
			QueueDepth      int `json:"queue_depth"`
			DeadLetterCount int `json:"dead_letter_count"`
		} `json:"extractor"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout.String())
	}
	if got.Extractor.QueueDepth != 0 {
		t.Errorf("extractor.queue_depth = %d, want 0 on fresh install", got.Extractor.QueueDepth)
	}
	if got.Extractor.DeadLetterCount != 0 {
		t.Errorf("extractor.dead_letter_count = %d, want 0 on fresh install", got.Extractor.DeadLetterCount)
	}
}

func TestMemoryStatusCommand_PrintsGonchoQueueZeroState(t *testing.T) {
	seedMemoryStatusDB(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Goncho queue status (observability/debugging only; not synchronization; do not wait for empty queue)",
		"representation: total=0 pending=0 in_progress=0 completed=0",
		"summary: total=0 pending=0 in_progress=0 completed=0",
		"dream: total=0 pending=0 in_progress=0 completed=0",
		"goncho_queue: unavailable (zero tracked work units)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want substring %q", out, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestMemoryStatusCommand_JSONEmitsStructuredSnapshot proves
// `gormes memory status --json` emits a parseable
// `{build, extractor: {...}, goncho_queue: {...}}` document. SREs
// monitoring fleet memory backlog (worker_health, queue_depth,
// dead_letter_count) need a structured shape for alerting; scraping
// the multi-section human format is fragile.
func TestMemoryStatusCommand_JSONEmitsStructuredSnapshot(t *testing.T) {
	seedMemoryStatusDB(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("memory status --json: %v\nstderr=%s", err, stderr.String())
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Extractor struct {
			WorkerHealth    string `json:"worker_health"`
			QueueDepth      int    `json:"queue_depth"`
			DeadLetterCount int    `json:"dead_letter_count"`
		} `json:"extractor"`
		GonchoQueue struct {
			ObservabilityOnly bool `json:"observability_only"`
		} `json:"goncho_queue"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if got.Extractor.WorkerHealth == "" {
		t.Fatalf("extractor.worker_health must be populated; got %+v", got.Extractor)
	}
	if got.Extractor.QueueDepth < 1 {
		t.Fatalf("extractor.queue_depth = %d, want >= 1 (fixture seeds at least one queued turn)", got.Extractor.QueueDepth)
	}
	if got.Extractor.DeadLetterCount != 2 {
		t.Fatalf("extractor.dead_letter_count = %d, want 2 (fixture seeds 2 dead letters)", got.Extractor.DeadLetterCount)
	}
	if !got.GonchoQueue.ObservabilityOnly {
		t.Fatalf("goncho_queue.observability_only = false, want true (matches goncho doctor convention)")
	}
	// JSON mode must not interleave the human "Extractor status" header.
	if strings.Contains(stdout.String(), "Extractor status") {
		t.Fatalf("--json must not emit the human header; got:\n%s", stdout.String())
	}
}

func TestMemoryStatusCommand_JSONEmitsProfileAwareInventory(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	gormesHome := filepath.Join(dataHome, "gormes")
	t.Setenv("GORMES_HOME", gormesHome)

	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "memory", "USER.md"), "native user memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "memories", "USER.md"), "legacy Hermes user memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "SOUL.md"), "profile soul\n")
	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "sessions", "index.yaml"), "sessions: {}\n")

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("memory status --json: %v\nstderr=%s", err, stderr.String())
	}

	var got struct {
		Inventory struct {
			SelectedPromptMemoryDir string `json:"selected_prompt_memory_dir"`
			LegacyImportNeeded      bool   `json:"legacy_import_needed"`
			Goncho                  struct {
				ActiveItems int `json:"active_items"`
			} `json:"goncho"`
			DurableMarkdown struct {
				User struct {
					State string `json:"state"`
				} `json:"user"`
			} `json:"durable_markdown"`
			LegacyHermes struct {
				User struct {
					State string `json:"state"`
				} `json:"user"`
			} `json:"legacy_hermes"`
			ContextFiles []struct {
				RelativePath string `json:"relative_path"`
				State        string `json:"state"`
			} `json:"context_files"`
			SessionTranscripts struct {
				Files int `json:"files"`
			} `json:"session_transcripts"`
		} `json:"inventory"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout.String())
	}
	if got.Inventory.Goncho.ActiveItems != 0 {
		t.Fatalf("inventory.goncho.active_items = %d, want 0 for empty Goncho DB", got.Inventory.Goncho.ActiveItems)
	}
	if got.Inventory.DurableMarkdown.User.State != "present" {
		t.Fatalf("durable_markdown.user.state = %q, want present", got.Inventory.DurableMarkdown.User.State)
	}
	if got.Inventory.LegacyHermes.User.State != "present" {
		t.Fatalf("legacy_llm.user.state = %q, want present", got.Inventory.LegacyHermes.User.State)
	}
	if got.Inventory.SelectedPromptMemoryDir != "memory" {
		t.Fatalf("selected_prompt_memory_dir = %q, want memory", got.Inventory.SelectedPromptMemoryDir)
	}
	if got.Inventory.LegacyImportNeeded {
		t.Fatal("legacy_import_needed = true, want false because native durable markdown exists")
	}
	if got.Inventory.SessionTranscripts.Files != 1 {
		t.Fatalf("session_transcripts.files = %d, want 1", got.Inventory.SessionTranscripts.Files)
	}
	if !memoryStatusContextFilePresent(got.Inventory.ContextFiles, "SOUL.md") {
		t.Fatalf("context_files = %+v, want present SOUL.md evidence", got.Inventory.ContextFiles)
	}
}

func TestMemoryStatusCommand_HumanOutputDoesNotFlattenDurableMemoryIntoZeroGonchoItems(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	gormesHome := filepath.Join(dataHome, "gormes")
	t.Setenv("GORMES_HOME", gormesHome)

	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "memory", "USER.md"), "native user memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "memories", "MEMORY.md"), "legacy Hermes memory\n")
	mustWriteInventoryTestFile(t, filepath.Join(gormesHome, "sessions", "index.yaml"), "sessions: {}\n")

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"memory", "status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("memory status: %v\nstderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Memory inventory",
		"goncho.active_items: 0",
		"durable_markdown_user: present",
		"legacy_hermes_memory: present",
		"selected_prompt_memory_dir: memory",
		"session_transcripts: files=1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func memoryStatusContextFilePresent(items []struct {
	RelativePath string `json:"relative_path"`
	State        string `json:"state"`
}, rel string) bool {
	for _, item := range items {
		if item.RelativePath == rel && item.State == "present" {
			return true
		}
	}
	return false
}

func mustWriteInventoryTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func seedMemoryStatusDB(t *testing.T) {
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

	now := time.Date(2026, 4, 22, 15, 4, 5, 0, time.UTC).Unix()
	_, err = store.DB().Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, extracted, extraction_attempts, extraction_error, cron)
		 VALUES
		 ('sess-1', 'user', 'queued turn', ?, 'telegram:1', 0, 0, NULL, 0),
		 ('sess-2', 'user', 'dead letter one', ?, 'telegram:2', 2, 3, 'malformed JSON', 0),
		 ('sess-3', 'assistant', 'dead letter two', ?, 'discord:9', 2, 4, 'upstream timeout', 0)`,
		now, now+1, now+2,
	)
	if err != nil {
		t.Fatalf("seed turns: %v", err)
	}
}
