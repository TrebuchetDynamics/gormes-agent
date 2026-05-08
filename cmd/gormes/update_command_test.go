package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestUpdateCommandRegistersNativeUpdate(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "update", "--help")
	if err != nil {
		t.Fatalf("update --help: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{"--branch", "--check", "--yes", "--restart-gateway", "--kill-stale-dashboard"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("update help missing %q:\n%s", want, stdout)
		}
	}
}

// TestUpdateCommand_HelpDocumentsRollbackWorkflow proves the long
// description of `gormes update --help` introduces the backup-restore
// rollback loop. Operators who never read the CHANGELOG should still
// discover that --backup pairs with `gormes restore --latest` for
// recovery.
func TestUpdateCommand_HelpDocumentsRollbackWorkflow(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeOneshotFlagCommand(cmd, "update", "--help")
	if err != nil {
		t.Fatalf("update --help: %v", err)
	}
	for _, want := range []string{
		"--backup",
		"gormes restore",
		"--latest",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("update --help long description must mention %q so the rollback workflow is discoverable; got:\n%s", want, stdout)
		}
	}
}

func TestUpdateCommandUsesInjectedLifecycle(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	var got cli.UpdateLifecycleOptions
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(ctx context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			if ctx == nil {
				t.Fatal("RunLifecycle got nil context")
			}
			return cli.UpdateReport{
				Branch: "development",
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceAutostashRestored, Detail: "ok"},
				},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--branch", "development", "--restart-gateway", "always")
	if err != nil {
		t.Fatalf("update command: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if got.CheckoutDir != "/repo/gormes" || got.Branch != "development" || got.RestartGateway != "always" {
		t.Fatalf("options = %+v, want checkout/branch/restart flags", got)
	}
	if !strings.Contains(stdout, "update branch: development") || !strings.Contains(stdout, "update_autostash_restored") {
		t.Fatalf("stdout missing summary/evidence:\n%s", stdout)
	}
}

// TestUpdateCommand_JSONEmitsReport proves `gormes update --json`
// replaces the human-readable print path with a machine-readable JSON
// document containing branch, evidence array, failed flag, and any
// operator_recovery hint. CI/cron consumers can parse the result
// without scraping ANSI-styled stdout.
func TestUpdateCommand_JSONEmitsReport(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(_ context.Context, _ cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{
				Branch:         "main",
				PreviousBranch: "main",
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceAutostashCreated, Detail: "stashed local changes"},
					{Kind: cli.UpdateEvidenceGatewayRestarted, Detail: "pid 12345"},
				},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--json")
	if err != nil {
		t.Fatalf("update --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}

	var got struct {
		Branch         string `json:"branch"`
		PreviousBranch string `json:"previous_branch"`
		Failed         bool   `json:"failed"`
		Evidence       []struct {
			Kind   string `json:"kind"`
			Detail string `json:"detail"`
		} `json:"evidence"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Branch != "main" {
		t.Fatalf("got.Branch = %q, want main", got.Branch)
	}
	if got.Failed {
		t.Fatalf("got.Failed = true, want false on success")
	}
	if len(got.Evidence) != 2 {
		t.Fatalf("got %d evidence entries, want 2", len(got.Evidence))
	}
	if got.Evidence[0].Kind != "update_autostash_created" {
		t.Fatalf("got.Evidence[0].Kind = %q, want `update_autostash_created`", got.Evidence[0].Kind)
	}
	// JSON mode must not interleave the human-readable banner.
	if strings.Contains(stdout, "⚕ Updating Gormes Agent") {
		t.Fatalf("--json must not emit the human banner; got:\n%s", stdout)
	}
}

// TestUpdateCommand_JSONOnFailureReturnsExitNonZero proves --json still
// honors the exit-code contract: a failed lifecycle exits non-zero so
// CI/cron `if cmd; then` checks work, while the JSON body on stdout
// remains parseable.
func TestUpdateCommand_JSONOnFailureReturnsExitNonZero(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{
				Branch: "main",
				Failed: true,
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceNetworkError, Detail: "dial tcp: connection refused"},
				},
				OperatorRecovery: "check network and retry",
			}
		},
	})

	stdout, _, err := executeRootCommandForTest(command, "--json")
	if err == nil {
		t.Fatalf("failed update with --json must return non-nil error; stdout=%s", stdout)
	}
	if code := exitCodeFromError(err); code == 0 {
		t.Fatalf("failed update --json exit code = 0, want non-zero")
	}
	var got struct {
		Failed           bool   `json:"failed"`
		OperatorRecovery string `json:"operator_recovery"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if !got.Failed {
		t.Fatalf("got.Failed = false, want true on failure")
	}
	if !strings.Contains(got.OperatorRecovery, "network") {
		t.Fatalf("got.OperatorRecovery = %q, want it to surface the recovery hint", got.OperatorRecovery)
	}
}

func TestUpdateCommandFailureReturnsOperatorEvidence(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{
				Branch: "development",
				Failed: true,
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceNetworkError, Detail: "could not fetch origin/development"},
					{Kind: cli.UpdateEvidenceAutostashPreserved, Detail: "stash preserved"},
				},
				OperatorRecovery: "Restore manually with: git stash apply stash-commit-safe",
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--branch", "development")
	if err == nil {
		t.Fatalf("update failure returned nil error; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"update_network_error", "update_autostash_preserved", "git stash apply"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("combined output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
}

// TestResolveBackupKeep proves the retention-budget resolver: positive
// values flow through from `[updates] backup_keep`, while 0/negative/
// missing config all fall back to defaultBackupKeep (5). This is the
// surface defaultBackupWriterFor reads to size the post-write prune.
func TestResolveBackupKeep(t *testing.T) {
	cases := []struct {
		name       string
		tomlBody   string
		wantKeep   int
		writeFile  bool
	}{
		{name: "no config file", tomlBody: "", wantKeep: 5, writeFile: false},
		{name: "missing updates section", tomlBody: "[hermes]\nmodel = \"x\"\n", wantKeep: 5, writeFile: true},
		{name: "explicit 7", tomlBody: "[updates]\nbackup_keep = 7\n", wantKeep: 7, writeFile: true},
		{name: "zero falls back", tomlBody: "[updates]\nbackup_keep = 0\n", wantKeep: 5, writeFile: true},
		{name: "negative falls back", tomlBody: "[updates]\nbackup_keep = -3\n", wantKeep: 5, writeFile: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", cfgHome)
			t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
			if tc.writeFile {
				dir := filepath.Join(cfgHome, "gormes")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tc.tomlBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := resolveBackupKeep()
			if got != tc.wantKeep {
				t.Fatalf("resolveBackupKeep() = %d, want %d", got, tc.wantKeep)
			}
		})
	}
}

// TestUpdateCommandReadsConfigBackupOptIn proves the cmd-side wiring
// loads `[updates] pre_update_backup = true` from config.toml and passes
// it to the lifecycle as BackupConfigEnabled. Without this, operators
// who set the config value would still need to pass --backup on every
// update run.
func TestUpdateCommandReadsConfigBackupOptIn(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[updates]
pre_update_backup = true
backup_keep = 7
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var got cli.UpdateLifecycleOptions
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(_ context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			return cli.UpdateReport{Branch: "main"}
		},
	})

	if _, stderr, err := executeRootCommandForTest(command, "--check"); err != nil {
		t.Fatalf("update --check: %v stderr=%s", err, stderr)
	}
	if !got.BackupConfigEnabled {
		t.Fatalf("BackupConfigEnabled = false, want true from `[updates] pre_update_backup = true`; got=%+v", got)
	}
}

func TestUpdateCommandCheckModeSkipsMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	called := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			called = true
			return cli.UpdateReport{}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--check", "--branch", "development")
	if err != nil {
		t.Fatalf("update --check: %v stderr=%s", err, stderr)
	}
	if !called {
		t.Fatal("RunLifecycle was not called")
	}
	if !strings.Contains(stdout, "update_check") {
		t.Fatalf("stdout missing check evidence:\n%s", stdout)
	}
}
