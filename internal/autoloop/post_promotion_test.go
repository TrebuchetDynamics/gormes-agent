package autoloop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTruncateLedgerDetail_PreservesTail verifies that long detail strings
// keep the END (where `go test ./...` prints --- FAIL summaries) instead of
// the head (which is just a list of passing packages). The pre-improvement
// truncation kept the first 2000 bytes and dropped the actual failure.
func TestTruncateLedgerDetail_PreservesTail(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("ok  \tgithub.com/Trebuchet/package/")
		b.WriteString(strings.Repeat("p", 12))
		b.WriteString("\t0.05s\n")
	}
	b.WriteString("--- FAIL: TestThatActuallyBroke (0.04s)\n")
	b.WriteString("    foo_test.go:42: expected 1, got 2\n")
	b.WriteString("FAIL\nFAIL\tgithub.com/Trebuchet/package/foo\t0.04s\n")
	b.WriteString("FAIL\n")
	full := b.String()
	if len(full) <= 2000 {
		t.Fatalf("synth input too short to exercise truncation (%d bytes)", len(full))
	}

	got := truncateLedgerDetail(full)
	if !strings.Contains(got, "--- FAIL: TestThatActuallyBroke") {
		t.Fatalf("truncated detail missing FAIL marker (tail must be preserved):\n%s", got)
	}
	if !strings.Contains(got, "FAIL\tgithub.com/Trebuchet/package/foo") {
		t.Fatalf("truncated detail missing FAIL summary line:\n%s", got)
	}
	if !strings.Contains(got, "bytes elided") {
		t.Fatalf("truncated detail missing elision marker:\n%s", got)
	}
	if len(got) > 2200 {
		t.Fatalf("truncated detail too long: %d bytes", len(got))
	}
}

func TestTruncateLedgerDetail_ShortStringPassesThrough(t *testing.T) {
	short := "command failed: exit 1"
	if got := truncateLedgerDetail(short); got != short {
		t.Fatalf("short string was truncated: got %q, want %q", got, short)
	}
}

// TestRunPostPromotionVerification_RunsAllCommandsAndCollectsFailures
// verifies that the verify gate does NOT abort on the first failed command.
// Recent ledger evidence shows verify aborts at command 1/5, so the operator
// (and the repair agent) never see whether commands 2-5 had additional
// problems. With the all-commands fix, a single verify failure event
// reports every broken command in one detail.
func TestRunPostPromotionVerification_RunsAllCommandsAndCollectsFailures(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		RepoRoot: dir,
		RunRoot:  filepath.Join(dir, "runroot"),
		PostPromotionVerifyCommands: []string{
			"true",
			"false",
			"true",
			"false",
			"true",
		},
	}

	calls := 0
	runner := runnerFunc(func(_ context.Context, cmd Command) Result {
		calls++
		// Simulate the shell-c failure for `false` commands.
		if strings.Contains(strings.Join(cmd.Args, " "), "false") {
			return Result{Err: errors.New("exit status 1"), Stderr: "boom"}
		}
		return Result{}
	})

	err := runPostPromotionVerification(context.Background(), cfg, runner, "test-run", 1)
	if err == nil {
		t.Fatal("expected verification error when 2 commands fail")
	}
	if calls != 5 {
		t.Fatalf("runner called %d times, want 5 (all commands must run)", calls)
	}

	// Read the ledger; assert ONE post_promotion_verify_failed event with a
	// detail that names BOTH failed commands.
	ledgerPath := filepath.Join(cfg.RunRoot, "state", "runs.jsonl")
	body := readLedgerLines(t, ledgerPath)
	var failures int
	var failedDetail string
	for _, line := range body {
		if strings.Contains(line, "post_promotion_verify_failed") {
			failures++
			failedDetail = line
		}
	}
	if failures != 1 {
		t.Fatalf("expected exactly 1 failure ledger event, got %d", failures)
	}
	if !strings.Contains(failedDetail, "command=2/5") {
		t.Fatalf("failure detail missing command 2/5 reference:\n%s", failedDetail)
	}
	if !strings.Contains(failedDetail, "command=4/5") {
		t.Fatalf("failure detail missing command 4/5 reference (all-failures collected):\n%s", failedDetail)
	}
}

// TestRunPrePromotionVerify_DisabledByDefaultIsNoop confirms that an empty
// PrePromotionVerifyCommands does not run any commands and emits no ledger
// events. This preserves the existing post-promotion-only behavior for
// installs that have not opted in.
func TestRunPrePromotionVerify_DisabledByDefaultIsNoop(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		RepoRoot: dir,
		RunRoot:  filepath.Join(dir, "runroot"),
	}
	calls := 0
	runner := runnerFunc(func(_ context.Context, _ Command) Result {
		calls++
		return Result{}
	})
	worker := workerRun{ID: 1, Task: "phase/sub/item", Branch: "autoloop/test/w1", RepoRoot: filepath.Join(dir, "worktree-1")}

	if err := runPrePromotionVerify(context.Background(), cfg, runner, "run-A", worker); err != nil {
		t.Fatalf("disabled gate should not error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("disabled gate must not run commands; got %d calls", calls)
	}
	if got := readLedgerLines(t, filepath.Join(cfg.RunRoot, "state", "runs.jsonl")); len(got) != 0 {
		t.Fatalf("disabled gate must not emit ledger events; got %d:\n%s", len(got), strings.Join(got, "\n"))
	}
}

// TestRunPrePromotionVerify_RunsInWorkerWorktree checks that the verify
// commands' Dir is set to worker.RepoRoot, NOT cfg.RepoRoot. This is the
// load-bearing distinction that keeps main from being briefly broken.
func TestRunPrePromotionVerify_RunsInWorkerWorktree(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		RepoRoot:                   filepath.Join(dir, "main"),
		RunRoot:                    filepath.Join(dir, "runroot"),
		PrePromotionVerifyCommands: []string{"echo ok"},
	}
	worktreePath := filepath.Join(dir, "worktree-1")
	worker := workerRun{ID: 1, Task: "phase/sub/item", Branch: "autoloop/test/w1", RepoRoot: worktreePath}

	var seenDir string
	runner := runnerFunc(func(_ context.Context, cmd Command) Result {
		seenDir = cmd.Dir
		return Result{}
	})

	if err := runPrePromotionVerify(context.Background(), cfg, runner, "run-A", worker); err != nil {
		t.Fatalf("verify should pass: %v", err)
	}
	if seenDir != worktreePath {
		t.Fatalf("Command.Dir = %q, want worker.RepoRoot %q (gate must run in worktree, not main)", seenDir, worktreePath)
	}
}

// TestRunPrePromotionVerify_FailureEmitsWorkerFailedAndPreventsPromotion is
// the headline behavior: a verify failure aborts the worker as a
// pre_promotion_verify_failed worker_failed event AND runs every command so
// the operator sees the full failure set in one ledger entry. The caller
// (finishWorker) bails on the returned error before promoteWorkerCommit.
func TestRunPrePromotionVerify_FailureEmitsWorkerFailedAndPreventsPromotion(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		RepoRoot:                   filepath.Join(dir, "main"),
		RunRoot:                    filepath.Join(dir, "runroot"),
		PrePromotionVerifyCommands: []string{"true", "false", "false"},
	}
	worker := workerRun{ID: 7, Task: "2/2.B/test-row", Branch: "autoloop/run-A/w7", RepoRoot: filepath.Join(dir, "worktree-7")}

	calls := 0
	runner := runnerFunc(func(_ context.Context, cmd Command) Result {
		calls++
		if strings.Contains(strings.Join(cmd.Args, " "), "false") {
			return Result{Err: errors.New("exit status 1"), Stderr: "boom"}
		}
		return Result{}
	})

	err := runPrePromotionVerify(context.Background(), cfg, runner, "run-A", worker)
	if err == nil {
		t.Fatal("verify failure must propagate as error")
	}
	if calls != 3 {
		t.Fatalf("all commands must run regardless of order; got %d calls, want 3", calls)
	}

	body := readLedgerLines(t, filepath.Join(cfg.RunRoot, "state", "runs.jsonl"))
	var startedSeen, failedSeen bool
	var failedDetail string
	for _, line := range body {
		if strings.Contains(line, `"event":"pre_promotion_verify_started"`) {
			startedSeen = true
		}
		if strings.Contains(line, `"event":"worker_failed"`) && strings.Contains(line, `"status":"pre_promotion_verify_failed"`) {
			failedSeen = true
			failedDetail = line
		}
	}
	if !startedSeen {
		t.Errorf("pre_promotion_verify_started event missing")
	}
	if !failedSeen {
		t.Fatalf("worker_failed/pre_promotion_verify_failed event missing:\n%s", strings.Join(body, "\n"))
	}
	for _, want := range []string{`"worker":7`, `"task":"2/2.B/test-row"`, "command=2/3", "command=3/3"} {
		if !strings.Contains(failedDetail, want) {
			t.Errorf("worker_failed event missing %q\n%s", want, failedDetail)
		}
	}
}

func TestRunPostPromotionVerification_AllCommandsPassEmitsSuccess(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		RepoRoot:                    dir,
		RunRoot:                     filepath.Join(dir, "runroot"),
		PostPromotionVerifyCommands: []string{"true", "true"},
	}
	runner := runnerFunc(func(_ context.Context, _ Command) Result { return Result{} })

	if err := runPostPromotionVerification(context.Background(), cfg, runner, "ok-run", 1); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	ledgerPath := filepath.Join(cfg.RunRoot, "state", "runs.jsonl")
	body := readLedgerLines(t, ledgerPath)
	var succeeded bool
	for _, line := range body {
		if strings.Contains(line, "post_promotion_verify_succeeded") {
			succeeded = true
		}
	}
	if !succeeded {
		t.Fatalf("expected post_promotion_verify_succeeded event in ledger:\n%s", strings.Join(body, "\n"))
	}
}

// readLedgerLines reads runs.jsonl as one line per event, returning a
// slice. Returns nil on missing file (treats as empty).
func readLedgerLines(t *testing.T, path string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read ledger %s: %v", path, err)
	}
	if len(body) == 0 {
		return nil
	}
	return strings.Split(strings.TrimRight(string(body), "\n"), "\n")
}
