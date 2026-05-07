package internal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCodexuBuilderLoopStatusReportsPauseState(t *testing.T) {
	repoRoot := testRepoRoot(t)
	stateDir := t.TempDir()
	script := filepath.Join(repoRoot, "scripts", "codexu-gormes-builder-loop.sh")

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

	script := filepath.Join(repoRoot, "scripts", "codexu-gormes-builder-loop.sh")
	cmd := exec.Command("timeout", "10s", "bash", script, "run")
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

	script := filepath.Join(repoRoot, "scripts", "codexu-gormes-builder-loop.sh")
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
		_ = os.WriteFile(filepath.Join(stateDir, "release-runner"), []byte("1\n"), 0o644)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if runnerPID != 0 {
			waitForPIDExit(t, runnerPID, 2*time.Second)
		}
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
