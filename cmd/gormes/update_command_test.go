package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
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

func TestUpdateCommandWiresBinaryPublisher(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	checkout := "/repo/gormes"
	publisherFactoryCalled := false
	publisherCalled := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return checkout, nil },
		BinaryPublisherFor: func(checkoutDir string) cli.UpdateBinaryPublisher {
			publisherFactoryCalled = true
			if checkoutDir != checkout {
				t.Fatalf("BinaryPublisherFor checkoutDir = %q, want %q", checkoutDir, checkout)
			}
			return func(_ context.Context, req cli.UpdateBinaryPublishRequest) cli.UpdateReport {
				publisherCalled = true
				if req.CheckoutDir != checkout || req.Branch != "development" {
					t.Fatalf("publish request = %+v, want checkout+branch", req)
				}
				return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{Kind: cli.UpdateEvidencePublishCompleted, Detail: "published"}}}
			}
		},
		RunLifecycle: func(ctx context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			if opts.BinaryPublisher == nil {
				t.Fatal("BinaryPublisher was not wired into lifecycle options")
			}
			return opts.BinaryPublisher(ctx, cli.UpdateBinaryPublishRequest{
				CheckoutDir: opts.CheckoutDir,
				Branch:      opts.Branch,
			})
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--branch", "development")
	if err != nil {
		t.Fatalf("update command: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !publisherFactoryCalled || !publisherCalled {
		t.Fatalf("publisherFactoryCalled=%v publisherCalled=%v", publisherFactoryCalled, publisherCalled)
	}
	if !strings.Contains(stdout, "update_publish_completed") {
		t.Fatalf("stdout missing publish evidence:\n%s", stdout)
	}
}

func TestUpdateCommandWiresGatewayRestartRunner(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	restartFactoryCalled := false
	restartCalled := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		GatewayRestartFor: func() cli.UpdateGatewayRestartRunner {
			restartFactoryCalled = true
			return func(_ context.Context, req cli.UpdateGatewayRestartRequest) cli.UpdateReport {
				restartCalled = true
				if req.Policy != "always" {
					t.Fatalf("restart policy = %q, want always", req.Policy)
				}
				return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{Kind: cli.UpdateEvidenceGatewayRestarted, Detail: "restarted"}}}
			}
		},
		RunLifecycle: func(ctx context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			if opts.GatewayRestart == nil {
				t.Fatal("GatewayRestart was not wired into lifecycle options")
			}
			return opts.GatewayRestart(ctx, cli.UpdateGatewayRestartRequest{Policy: opts.RestartGateway})
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--restart-gateway", "always")
	if err != nil {
		t.Fatalf("update command: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !restartFactoryCalled || !restartCalled {
		t.Fatalf("restartFactoryCalled=%v restartCalled=%v", restartFactoryCalled, restartCalled)
	}
	if !strings.Contains(stdout, "update_gateway_restarted") {
		t.Fatalf("stdout missing restart evidence:\n%s", stdout)
	}
}

func TestUpdateCommandWritesUpdateLogAndLedgerForRealUpdate(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	installHome := t.TempDir()
	t.Setenv("GORMES_INSTALL_HOME", installHome)
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{
				Branch:         "development",
				PreviousBranch: "development",
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidencePublishCompleted, Detail: "published"},
				},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--branch", "development", "--restart-gateway", "never")
	if err != nil {
		t.Fatalf("update: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{"update_hangup_log_mirrored", "update_ledger_appended"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	logBody, err := os.ReadFile(filepath.Join(installHome, "update.log"))
	if err != nil {
		t.Fatalf("read update.log: %v", err)
	}
	if !strings.Contains(string(logBody), "update_publish_completed") || !strings.Contains(string(logBody), "update_ledger_appended") {
		t.Fatalf("update.log missing mirrored update report:\n%s", logBody)
	}
	ledgerBody, err := os.ReadFile(filepath.Join(installHome, "install.log.jsonl"))
	if err != nil {
		t.Fatalf("read install.log.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(ledgerBody)), "\n")
	if len(lines) != 1 {
		t.Fatalf("ledger line count = %d, want 1:\n%s", len(lines), ledgerBody)
	}
	var event struct {
		Event          string `json:"event"`
		Branch         string `json:"branch"`
		PreviousBranch string `json:"previous_branch"`
		RestartGateway string `json:"restart_gateway"`
		Failed         bool   `json:"failed"`
		Evidence       []struct {
			Kind string `json:"kind"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("ledger line is not valid JSON: %v\n%s", err, lines[0])
	}
	if event.Event != "update" || event.Branch != "development" || event.PreviousBranch != "development" || event.RestartGateway != "never" || event.Failed {
		t.Fatalf("ledger event = %+v, want update/development/never/success", event)
	}
	if len(event.Evidence) == 0 || event.Evidence[0].Kind != "update_publish_completed" {
		t.Fatalf("ledger evidence = %+v, want update_publish_completed first", event.Evidence)
	}
}

func TestDefaultUpdateBinaryPublishOptionsRespectSandboxBinDir(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	home := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	t.Setenv("GORMES_INSTALL_HOME", home)
	t.Setenv("GORMES_BIN_DIR", binDir)
	t.Setenv("GORMES_PREFIX", "")

	opts, err := defaultUpdateBinaryPublishOptions("/repo/gormes")
	if err != nil {
		t.Fatalf("defaultUpdateBinaryPublishOptions: %v", err)
	}
	if opts.ManagedBinPath != filepath.Join(home, "bin", "gormes") {
		t.Fatalf("ManagedBinPath = %q, want managed home bin", opts.ManagedBinPath)
	}
	if opts.PublishedBinPath != filepath.Join(binDir, "gormes") {
		t.Fatalf("PublishedBinPath = %q, want GORMES_BIN_DIR target", opts.PublishedBinPath)
	}
	if opts.RefreshActivePath {
		t.Fatalf("RefreshActivePath = true, want false inside GORMES_BIN_DIR sandbox")
	}
}

func TestUpdateCommand_PrintsCuratorRecentRunSummaryOnce(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := filepath.Join(os.Getenv("GORMES_HOME"), "skills")
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatalf("mkdir skills root: %v", err)
	}
	lastRun := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Second)
	summary := "auto: no changes; llm: consolidated 1 into 1\n" +
		"archived 1 skill(s):\n" +
		"  • old-skill → new-skill\n" +
		"full report: gormes curator status"
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:      &lastRun,
		LastRunSummary: summary,
		RunCount:       1,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{Branch: "main"}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command)
	if err != nil {
		t.Fatalf("update: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"Skill curator",
		"last run",
		"old-skill → new-skill",
		"shows once per curator run",
		"gormes curator status",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("update stdout missing curator notice %q:\n%s", want, stdout)
		}
	}
	state, err := curator.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.LastRunSummaryShownAt == nil || !state.LastRunSummaryShownAt.Equal(lastRun) {
		t.Fatalf("LastRunSummaryShownAt = %v, want %s", state.LastRunSummaryShownAt, lastRun)
	}

	command = newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{Branch: "main"}
		},
	})
	stdout, stderr, err = executeRootCommandForTest(command)
	if err != nil {
		t.Fatalf("second update: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if strings.Contains(stdout, "old-skill → new-skill") {
		t.Fatalf("curator notice printed twice:\n%s", stdout)
	}
}

func TestUpdateCommand_SkipsCuratorRecentRunSummaryNoopOrShown(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := filepath.Join(os.Getenv("GORMES_HOME"), "skills")
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatalf("mkdir skills root: %v", err)
	}
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	lastRun := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:      &lastRun,
		LastRunSummary: "auto: no changes; llm: no change",
		RunCount:       1,
	}); err != nil {
		t.Fatalf("SaveState noop: %v", err)
	}
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{Branch: "main"}
		},
	})
	stdout, stderr, err := executeRootCommandForTest(command)
	if err != nil {
		t.Fatalf("noop update: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if strings.Contains(stdout, "Skill curator") {
		t.Fatalf("single-line no-op summary should stay silent:\n%s", stdout)
	}
	state, err := curator.LoadState()
	if err != nil {
		t.Fatalf("LoadState noop: %v", err)
	}
	if state.LastRunSummaryShownAt == nil || !state.LastRunSummaryShownAt.Equal(lastRun) {
		t.Fatalf("noop LastRunSummaryShownAt = %v, want %s", state.LastRunSummaryShownAt, lastRun)
	}

	newer := time.Now().UTC().Truncate(time.Second)
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:              &newer,
		LastRunSummaryShownAt:  state.LastRunSummaryShownAt,
		LastRunSummary:         "auto: no changes; llm: consolidated 1 into 1\narchived 1 skill(s):\n  • newer → umbrella\nfull report: gormes curator status",
		RunCount:               2,
		LastRunDurationSeconds: state.LastRunDurationSeconds,
	}); err != nil {
		t.Fatalf("SaveState newer: %v", err)
	}
	command = newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{Branch: "main"}
		},
	})
	stdout, stderr, err = executeRootCommandForTest(command)
	if err != nil {
		t.Fatalf("newer update: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "newer → umbrella") {
		t.Fatalf("newer curator run should print after an older shown stamp:\n%s", stdout)
	}
}

// TestUpdateCommand_JSONIncludesBuildProvenance proves
// `gormes update --json` carries the running binary's build SHA and
// version so an operator inspecting a captured update report can
// correlate which gormes binary actually ran the lifecycle. Surfacing
// only the lifecycle outcome (`failed`, evidence) without the
// originating binary identity makes drift hard to attribute.
func TestUpdateCommand_JSONIncludesBuildProvenance(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(_ context.Context, _ cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{Branch: "main"}
		},
	})

	stdout, _, err := executeRootCommandForTest(command, "--check", "--json")
	if err != nil {
		t.Fatalf("update --check --json: %v", err)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q (matches package-level Version)", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty (`unknown` is acceptable for dev builds)")
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
	if len(got.Evidence) != 4 {
		t.Fatalf("got %d evidence entries, want lifecycle evidence plus log/ledger evidence", len(got.Evidence))
	}
	if got.Evidence[0].Kind != "update_autostash_created" {
		t.Fatalf("got.Evidence[0].Kind = %q, want `update_autostash_created`", got.Evidence[0].Kind)
	}
	for _, want := range []string{"update_hangup_log_mirrored", "update_ledger_appended"} {
		found := false
		for _, evidence := range got.Evidence {
			if evidence.Kind == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("JSON evidence missing %q: %+v", want, got.Evidence)
		}
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
		name      string
		tomlBody  string
		wantKeep  int
		writeFile bool
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

// TestUpdateCommand_DefaultCheckoutDirHonorsGormesInstallDir pins the
// safety guarantee that `gormes update` (with no seam override) targets
// the install's managed source clone, NOT whatever cwd happens to be a
// git worktree. Regression observed during a v0.2.0 fresh-install
// probe: running `gormes update` from inside the gormes-agent dev tree
// switched that tree's branch from `development` to `main` and ran a
// web build there — because the default CheckoutDir was `os.Getwd`.
//
// install.sh exposes the managed checkout location via
// `GORMES_INSTALL_DIR`. The default resolver MUST honor it so the
// runtime updates the install, not whatever directory the operator
// invoked the command from.
func TestUpdateCommand_DefaultCheckoutDirHonorsGormesInstallDir(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	expected := t.TempDir()
	t.Setenv("GORMES_INSTALL_DIR", expected)

	var got cli.UpdateLifecycleOptions
	command := newUpdateCommandWithSeams(updateCommandSeams{
		// Intentionally leave CheckoutDir nil so the production
		// resolver fires.
		RunLifecycle: func(_ context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			return cli.UpdateReport{Branch: "main"}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--check")
	if err != nil {
		t.Fatalf("update --check: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got.CheckoutDir != expected {
		t.Fatalf("CheckoutDir = %q, want %q (must come from GORMES_INSTALL_DIR, not os.Getwd())", got.CheckoutDir, expected)
	}
}

// TestUpdateCommand_DefaultCheckoutDirFallsBackToManagedHomeNotCwd
// pins the same safety guarantee for the no-env-var path: when
// GORMES_INSTALL_DIR is unset, the resolver returns
// `$GORMES_INSTALL_HOME/gormes-agent` (mirror of install.sh's
// managed_checkout_dir() default). Critical: cwd may be an unrelated
// git checkout (the developer's source tree, a CI workspace) — `gormes
// update` must never mutate it.
func TestUpdateCommand_DefaultCheckoutDirFallsBackToManagedHomeNotCwd(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	if err := os.Unsetenv("GORMES_INSTALL_DIR"); err != nil {
		t.Fatalf("unset GORMES_INSTALL_DIR: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("GORMES_INSTALL_DIR") })
	installHome := t.TempDir()
	t.Setenv("GORMES_INSTALL_HOME", installHome)
	expected := filepath.Join(installHome, "gormes-agent")

	var got cli.UpdateLifecycleOptions
	command := newUpdateCommandWithSeams(updateCommandSeams{
		RunLifecycle: func(_ context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			return cli.UpdateReport{Branch: "main"}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--check")
	if err != nil {
		t.Fatalf("update --check: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got.CheckoutDir != expected {
		t.Fatalf("CheckoutDir = %q, want %q (managed default `$GORMES_INSTALL_HOME/gormes-agent`, not os.Getwd())", got.CheckoutDir, expected)
	}
}
