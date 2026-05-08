package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTerminalToolDeletedProcessCWDRecoversToConfiguredCWD proves
// that when the process's own CWD has been deleted (os.Getwd()
// returns an error), the terminal tool recovers by using the
// configured terminal.cwd instead of falling back to os.TempDir().
func TestTerminalToolDeletedCWDRecoversToConfiguredCWD(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get original cwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	root := t.TempDir()
	configured := filepath.Join(root, "configured-workdir")
	if err := os.MkdirAll(configured, 0o755); err != nil {
		t.Fatalf("mkdir configured workdir: %v", err)
	}

	// Create a deleted directory and chdir into it, then delete it
	deleted := filepath.Join(root, "wedged")
	if err := os.MkdirAll(deleted, 0o755); err != nil {
		t.Fatalf("mkdir deleted cwd: %v", err)
	}
	if err := os.Chdir(deleted); err != nil {
		t.Fatalf("chdir deleted cwd: %v", err)
	}
	if err := os.RemoveAll(deleted); err != nil {
		t.Fatalf("remove current cwd: %v", err)
	}

	// The configured workdir still exists — the tool should recover to it
	tool := NewTerminalTool(TerminalToolConfig{Workdir: configured, DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"pwd"}`)

	if out["status"] != "completed" {
		t.Fatalf("status = %v, want completed: %#v", out["status"], out)
	}
	if out["workdir"] != configured {
		t.Fatalf("workdir = %v, want configured workdir %q: %#v", out["workdir"], configured, out)
	}
	if strings.TrimSpace(asString(out["output"])) != configured {
		t.Fatalf("output = %q, want pwd %q", out["output"], configured)
	}
	if out["cwd_recovered"] != true {
		t.Fatalf("cwd_recovered = %v, want true: %#v", out["cwd_recovered"], out)
	}
	if !strings.Contains(asString(out["cwd_recovery"]), "terminal_cwd_recovered") {
		t.Fatalf("cwd_recovery = %v, want terminal_cwd_recovered evidence", out["cwd_recovery"])
	}
}

// TestTerminalToolDeletedConfiguredCWDReturnsEvidence proves that
// when the explicitly-configured default CWD path has been deleted
// (not a placeholder like "." or "auto"), the terminal tool fails
// closed with terminal_cwd_deleted evidence rather than silently
// recovering to a parent directory. This prevents operators from
// unknowingly running commands from the wrong directory.
func TestTerminalToolDeletedConfiguredCWDReturnsEvidence(t *testing.T) {
	root := t.TempDir()
	configured := filepath.Join(root, "my-specific-project")
	// Create then delete so it's a real path that no longer exists
	if err := os.MkdirAll(configured, 0o755); err != nil {
		t.Fatalf("mkdir configured: %v", err)
	}
	if err := os.RemoveAll(configured); err != nil {
		t.Fatalf("remove configured: %v", err)
	}

	tool := NewTerminalTool(TerminalToolConfig{Workdir: configured, DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"pwd"}`)

	if out["status"] != "error" {
		t.Fatalf("status = %v, want error (fail closed): %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "terminal_cwd_deleted") {
		t.Fatalf("error = %v, want terminal_cwd_deleted evidence", out["error"])
	}
	if out["exit_code"] != float64(-1) {
		t.Fatalf("exit_code = %v, want -1", out["exit_code"])
	}
	// Must NOT have recovered — no cwd_recovered signal
	if _, ok := out["cwd_recovered"]; ok {
		t.Fatalf("cwd_recovered should not be present when failing closed: %#v", out)
	}
}

// TestTerminalBackgroundSpawnRejectsDeletedCWD proves that when
// background is requested and the CWD is deleted, the tool rejects
// the request BEFORE any subprocess creation with cwd-specific
// evidence, rather than the generic "background unsupported" message.
// This is a guard for future background process support.
func TestTerminalBackgroundSpawnRejectsDeletedCWD(t *testing.T) {
	root := t.TempDir()
	deleted := filepath.Join(root, "gone")
	if err := os.MkdirAll(deleted, 0o755); err != nil {
		t.Fatalf("mkdir deleted: %v", err)
	}
	if err := os.RemoveAll(deleted); err != nil {
		t.Fatalf("remove deleted: %v", err)
	}

	tool := NewTerminalTool(TerminalToolConfig{Workdir: deleted, DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"sleep 99","background":true}`)

	// Should reject due to deleted cwd, NOT due to generic background unsupported
	if out["status"] != "error" {
		t.Fatalf("status = %v, want error: %#v", out["status"], out)
	}
	errStr := asString(out["error"])
	if !strings.Contains(errStr, "terminal_cwd_deleted") {
		t.Fatalf("error = %v, want terminal_cwd_deleted evidence (not generic background rejection)", errStr)
	}
	if out["exit_code"] != float64(-1) {
		t.Fatalf("exit_code = %v, want -1", out["exit_code"])
	}
}

// TestPersistentShellDeletedCWDResetsState proves that the local
// environment wrapper (FakeEnvironment) clears stale CWD state
// when the working directory has been deleted, reporting recovery
// evidence. This mirrors Hermes' _update_cwd guard which rejects
// stale pwd-P output pointing to deleted directories.
func TestPersistentShellDeletedCWDResetsState(t *testing.T) {
	root := t.TempDir()
	deletedCWD := filepath.Join(root, "deleted-session-dir")
	if err := os.MkdirAll(deletedCWD, 0o755); err != nil {
		t.Fatalf("mkdir deletedCWD: %v", err)
	}
	if err := os.RemoveAll(deletedCWD); err != nil {
		t.Fatalf("remove deletedCWD: %v", err)
	}

	mapper := NewEnvironmentPathMapper(root, "/env")
	env := NewFakeEnvironment("local", mapper)

	// Simulate executing a command with a deleted CWD
	cmd := EnvironmentCommand{
		Command:    "echo hello",
		WorkingDir: deletedCWD,
		Timeout:    5 * time.Second,
	}
	result, err := env.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute should not raise error for deleted cwd: %v", err)
	}

	// Check that evidence shows cwd was detected as deleted and reset
	foundReset := false
	for _, ev := range result.Evidence {
		if ev.Code == EnvironmentTerminalCWDDeleted || strings.Contains(ev.Message, "cwd") {
			foundReset = true
			if ev.Operation != string(EnvironmentOperationExecute) {
				t.Errorf("cwd evidence operation = %q, want %q", ev.Operation, EnvironmentOperationExecute)
			}
			break
		}
	}
	if !foundReset {
		t.Errorf("expected cwd-reset evidence in result, got: %#v", result.Evidence)
	}

	// After the reset, subsequent Execute with the same deleted CWD
	// should also reset (not cache the stale value)
	result2, err := env.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	foundReset2 := false
	for _, ev := range result2.Evidence {
		if ev.Code == EnvironmentTerminalCWDDeleted || strings.Contains(ev.Message, "cwd") {
			foundReset2 = true
			break
		}
	}
	if !foundReset2 {
		t.Errorf("expected cwd-reset evidence on subsequent call too, got: %#v", result2.Evidence)
	}
}



