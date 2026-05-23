package internal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCodexuBuilderLoopStatusReportsPauseState(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")

	cmd := exec.Command("bash", script, "pause", "--ttl", "10m", "gormes-git waiting for active run")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(), "GORMES_CODEXU_STATE_DIR="+stateDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pause failed: %v\noutput:\n%s", err, string(out))
	}

	cmd = exec.Command("bash", script, "status")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(), "GORMES_CODEXU_STATE_DIR="+stateDir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"pause_file=" + filepath.Join(stateDir, "pause"),
		"pause_state=active",
		"pause_reason=gormes-git waiting for active run",
		"pause_expires_at=",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestCodexuBuilderLoopStatusReportsLiveProgressSignals(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	tmpRepo := filepath.Join(stateDir, "repo")
	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")

	writeFile(t, filepath.Join(tmpRepo, "README.md"), []byte("before\n"), 0o644)
	runCommand(t, tmpRepo, "git", "init")
	runCommand(t, tmpRepo, "git", "config", "user.name", "Test User")
	runCommand(t, tmpRepo, "git", "config", "user.email", "test@example.com")
	runCommand(t, tmpRepo, "git", "add", ".")
	runCommand(t, tmpRepo, "git", "commit", "-m", "init")
	writeFile(t, filepath.Join(tmpRepo, "README.md"), []byte("after\n"), 0o644)
	writeFile(t, filepath.Join(stateDir, "logs", "20260507T000000Z.log"), []byte("runner active\n"), 0o600)
	writeFile(t, filepath.Join(stateDir, "last-message.txt"), []byte("last message\n"), 0o600)

	cmd := exec.Command("bash", script, "status")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(),
		"GORMES_CODEXU_REPO="+tmpRepo,
		"GORMES_CODEXU_STATE_DIR="+stateDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"live progress:",
		"repo_dirty_state=dirty",
		"repo_dirty_count=1",
		"repo_ahead=unknown",
		"repo_behind=unknown",
		"latest_log=" + filepath.Join(stateDir, "logs", "20260507T000000Z.log"),
		"latest_log_size_bytes=14",
		"latest_log_age_seconds=",
		"last_message_file=" + filepath.Join(stateDir, "last-message.txt"),
		"last_message_age_seconds=",
		"current_run_state=absent",
		"last_success_state=absent",
		"last_failure_state=absent",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestCodexuBuilderLoopCostReportDoesNotFakeZeroWhenDataMissing(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")

	cmd := exec.Command("bash", script, "cost-report")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(), "GORMES_CODEXU_STATE_DIR="+stateDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cost-report failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"cost_status=unknown_no_cost_data",
		"cost_7d_usd=unknown",
		"cost_30d_usd=unknown",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("cost-report output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "7-day spend: $0.00") || strings.Contains(output, "30-day spend: $0.00") {
		t.Fatalf("cost-report must not present missing cost data as zero spend:\n%s", output)
	}
}

func TestCodexuBuilderLoopCostReportAcceptsOpenCodePartCostTelemetry(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")
	writeFile(t, filepath.Join(stateDir, "logs", "run-part.opencode.jsonl"), []byte(`{"type":"step_finish","timestamp":1778160185296,"sessionID":"ses-test","part":{"type":"step-finish","cost":0.08239596,"tokens":{"input":46206,"output":126}}}`+"\n"), 0o600)

	cmd := exec.Command("bash", script, "cost-report")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(), "GORMES_CODEXU_STATE_DIR="+stateDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cost-report failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"cost_status=ok",
		"cost_7d_usd=0.0824",
		"cost_30d_usd=0.0824",
		"cost_7d_runs=1",
		"cost_30d_runs=1",
		"7-day spend: $0.08 | 30-day spend: $0.08 | runs: 1 (7d) / 1 (30d)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("cost-report output missing %q:\n%s", want, output)
		}
	}
}

func TestCodexuBuilderLoopCostReportDoesNotFakeZeroForMissingSevenDayWindow(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")
	logFile := filepath.Join(stateDir, "logs", "run-part-old.opencode.jsonl")
	writeFile(t, logFile, []byte(`{"type":"step_finish","timestamp":1778160185296,"sessionID":"ses-test","part":{"type":"step-finish","cost":0.08239596,"tokens":{"input":46206,"output":126}}}`+"\n"), 0o600)
	oldEnoughForThirtyDaysOnly := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(logFile, oldEnoughForThirtyDaysOnly, oldEnoughForThirtyDaysOnly); err != nil {
		t.Fatalf("set log time: %v", err)
	}

	cmd := exec.Command("bash", script, "cost-report")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(), "GORMES_CODEXU_STATE_DIR="+stateDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cost-report failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"cost_status=ok",
		"cost_7d_usd=unknown",
		"cost_7d_runs=0",
		"cost_30d_usd=0.0824",
		"cost_30d_runs=1",
		"7-day spend: unknown | 30-day spend: $0.08 | runs: 0 (7d) / 1 (30d)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("cost-report output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "7-day spend: $0.00") || strings.Contains(output, "invalid number") {
		t.Fatalf("cost-report must keep unknown seven-day cost explicit without printf errors:\n%s", output)
	}
}

func TestCodexuBuilderLoopHealthIncludesCostTelemetry(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	homeDir := filepath.Join(stateDir, "home")
	if err := os.Mkdir(homeDir, 0o700); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	runner := filepath.Join(stateDir, "runner.sh")
	writeFile(t, runner, []byte(`#!/usr/bin/env bash
set -Eeuo pipefail
printf 'timestamp=%q\nreason=%q\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "cost telemetry test stop" > "$GORMES_CODEXU_STATE_DIR/stop-after-current"
`), 0o755)
	writeFile(t, filepath.Join(stateDir, "logs", "run-a.opencode.jsonl"), []byte(`{"usage":{"cost":0.0123,"input_tokens":1000,"output_tokens":200},"timestamp":"2026-05-23T00:00:00Z"}`+"\n"), 0o600)

	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")
	cmd := exec.Command("timeout", "30s", "bash", script, "run")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(),
		"GORMES_CODEXU_REPO="+repoRoot,
		"GORMES_CODEXU_RUNNER="+runner,
		"GORMES_CODEXU_STATE_DIR="+stateDir,
		"GORMES_CODEXU_LOOP_INTERVAL=0",
		"GORMES_CODEXU_PAUSE_POLL=1",
		"GORMES_CODEXU_FAIL_BACKOFF=1",
		"HOME="+homeDir,
		"PATH=/usr/local/bin:/usr/bin:/bin",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("loop run failed: %v\noutput:\n%s", err, string(out))
	}

	health := readOptionalFile(t, filepath.Join(stateDir, "loop-health.env"))
	for _, want := range []string{
		"cost_status=ok",
		"cost_7d_usd=0.0123",
		"cost_30d_usd=0.0123",
		"cost_7d_runs=1",
		"cost_30d_runs=1",
	} {
		if !strings.Contains(health, want) {
			t.Fatalf("loop-health.env missing %q:\n%s", want, health)
		}
	}
}

func TestCodexuBuilderLoopAutoClearsExpiredPauseBeforeRunner(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	homeDir := filepath.Join(stateDir, "home")
	if err := os.Mkdir(homeDir, 0o700); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	runner := filepath.Join(stateDir, "runner.sh")
	writeFile(t, runner, []byte(`#!/usr/bin/env bash
set -Eeuo pipefail
printf 'ran\n' >> "$GORMES_CODEXU_STATE_DIR/runner.log"
printf 'timestamp=%q\nreason=%q\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "test requested stop" > "$GORMES_CODEXU_STATE_DIR/stop-after-current"
`), 0o755)

	pause := filepath.Join(stateDir, "pause")
	expired := time.Now().Add(-time.Minute).Unix()
	if err := os.WriteFile(pause, []byte("timestamp=2026-05-07T00:36:41Z\nreason=stale gormes-git safety pause\nexpires_at_epoch="+strconv.FormatInt(expired, 10)+"\n"), 0o600); err != nil {
		t.Fatalf("write pause: %v", err)
	}

	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")
	cmd := exec.Command("timeout", "30s", "bash", script, "run")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(),
		"GORMES_CODEXU_REPO="+repoRoot,
		"GORMES_CODEXU_RUNNER="+runner,
		"GORMES_CODEXU_STATE_DIR="+stateDir,
		"GORMES_CODEXU_LOOP_INTERVAL=0",
		"GORMES_CODEXU_PAUSE_POLL=1",
		"GORMES_CODEXU_FAIL_BACKOFF=1",
		"HOME="+homeDir,
		"PATH=/usr/local/bin:/usr/bin:/bin",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("loop run failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "pause file expired; auto-resuming") {
		t.Fatalf("loop output missing auto-resume log:\n%s", output)
	}
	if _, err := os.Stat(pause); !os.IsNotExist(err) {
		t.Fatalf("pause file still present after expired auto-resume: %v", err)
	}
	if got := readOptionalFile(t, filepath.Join(stateDir, "runner.log")); got != "ran\n" {
		t.Fatalf("runner log = %q, want runner to execute once", got)
	}
}

func TestCodexuBuilderLoopDoesNotLeakLoopLockToRunner(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	runner := filepath.Join(stateDir, "runner.sh")
	writeFile(t, runner, []byte(`#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "$$" > "$GORMES_CODEXU_STATE_DIR/runner.pid"
while [ ! -f "$GORMES_CODEXU_STATE_DIR/release-runner" ]; do
  sleep 0.1
done
`), 0o755)

	script := filepath.Join(repoRoot, "scripts", "gormes-builder-loop.sh")
	cmd := exec.Command("bash", script, "run")
	cmd.Dir = repoRoot
	cmd.Env = overlayEnv(os.Environ(),
		"GORMES_CODEXU_REPO="+repoRoot,
		"GORMES_CODEXU_RUNNER="+runner,
		"GORMES_CODEXU_STATE_DIR="+stateDir,
		"GORMES_CODEXU_LOOP_INTERVAL=60",
		"GORMES_CODEXU_FAIL_BACKOFF=1",
	)
	outFile := filepath.Join(stateDir, "loop.out")
	out, err := os.Create(outFile)
	if err != nil {
		t.Fatalf("create loop output: %v", err)
	}
	defer out.Close()
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start loop: %v", err)
	}
	runnerPID := 0
	defer func() {
		// Cooperative path: write the sentinel so the runner's wait
		// loop exits cleanly within ~100ms.
		_ = os.WriteFile(filepath.Join(stateDir, "release-runner"), []byte("1\n"), 0o644)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if runnerPID == 0 {
			return
		}
		// Defensive path: SIGKILL the runner unconditionally before
		// the test's t.TempDir cleanup deletes its stateDir. Without
		// this, ANY failure to read the sentinel (state dir already
		// gone, slow write, race during t.Fatalf unwind) orphans the
		// runner — its `while [ ! -f $STATE/release-runner ]` loop
		// then spins forever against a missing dir, accumulating one
		// leaked process per failed test run.
		_ = syscall.Kill(runnerPID, syscall.SIGKILL)
		waitForPIDExit(t, runnerPID, 2*time.Second)
	}()

	runnerPIDPath := filepath.Join(stateDir, "runner.pid")
	waitForFile(t, runnerPIDPath, 5*time.Second)
	pidBytes, err := os.ReadFile(runnerPIDPath)
	if err != nil {
		t.Fatalf("read runner pid: %v", err)
	}
	runnerPID, err = strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		t.Fatalf("parse runner pid: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill loop process: %v", err)
	}
	_ = cmd.Wait()

	lockPath := filepath.Join(stateDir, "loop.lock")
	lockCmd := exec.Command("flock", "-n", lockPath, "true")
	if out, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("loop lock remained held after loop parent died; runner inherited it: %v\noutput:\n%s", err, string(out))
	}
}

func waitForPIDExit(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	procPath := filepath.Join("/proc", strconv.Itoa(pid))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(procPath); os.IsNotExist(err) {
			return
		} else if err != nil {
			t.Fatalf("stat %s: %v", procPath, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for pid %d to exit", pid)
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}
