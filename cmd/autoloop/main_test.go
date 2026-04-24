package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/autoloop"
)

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := run([]string{"unknown"})
	if err == nil {
		t.Fatal("run() error = nil, want error")
	}
	for _, want := range []string{
		"usage: autoloop",
		"audit",
		"service install",
		"service install-audit",
		"service disable legacy-timers",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("run() error = %q, want %q", err, want)
		}
	}
}

func TestRunCommandDryRunPrintsSummary(t *testing.T) {
	repoRoot := t.TempDir()
	progressPath := filepath.Join(repoRoot, "progress.json")
	if err := os.WriteFile(progressPath, []byte(`{
		"phases": {
			"12": {
				"subphases": {
					"12.A": {
						"items": [
							{"item_name": "planned CLI candidate", "status": "planned"}
						]
					}
				}
			}
		}
	}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("PROGRESS_JSON", progressPath)
	t.Setenv("RUN_ROOT", filepath.Join(repoRoot, "runs"))
	t.Setenv("BACKEND", "opencode")
	t.Setenv("MODE", "safe")
	t.Setenv("MAX_AGENTS", "1")

	var stdout bytes.Buffer
	oldStdout := commandStdout
	commandStdout = &stdout
	t.Cleanup(func() {
		commandStdout = oldStdout
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"run", "--dry-run"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"candidates: 1", "selected: 1", "planned CLI candidate"} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want %q", output, want)
		}
	}
}

func TestDigestUsesConfiguredRunRoot(t *testing.T) {
	repoRoot := t.TempDir()
	runRoot := filepath.Join(repoRoot, "custom-runs")
	ledgerPath := filepath.Join(runRoot, "state", "runs.jsonl")
	if err := autoloop.AppendLedgerEvent(ledgerPath, autoloop.LedgerEvent{
		TS:    time.Unix(1, 0).UTC(),
		Event: "run_started",
	}); err != nil {
		t.Fatalf("AppendLedgerEvent() error = %v", err)
	}
	t.Setenv("RUN_ROOT", runRoot)

	var stdout bytes.Buffer
	oldStdout := commandStdout
	commandStdout = &stdout
	t.Cleanup(func() {
		commandStdout = oldStdout
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"digest"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "runs: 1") {
		t.Fatalf("stdout = %q, want digest from configured RUN_ROOT", stdout.String())
	}
}

func TestAuditUsesConfiguredRunRoot(t *testing.T) {
	repoRoot := t.TempDir()
	runRoot := filepath.Join(repoRoot, "custom-runs")
	ledgerPath := filepath.Join(runRoot, "state", "runs.jsonl")
	if err := autoloop.AppendLedgerEvent(ledgerPath, autoloop.LedgerEvent{
		TS:    time.Unix(1, 0).UTC(),
		Event: "run_started",
	}); err != nil {
		t.Fatalf("AppendLedgerEvent() error = %v", err)
	}
	t.Setenv("RUN_ROOT", runRoot)

	var stdout bytes.Buffer
	oldStdout := commandStdout
	commandStdout = &stdout
	t.Cleanup(func() {
		commandStdout = oldStdout
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"audit"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "runs: 1") {
		t.Fatalf("stdout = %q, want audit digest from configured RUN_ROOT", stdout.String())
	}
}

func TestServiceInstallWritesUnitUnderXDGConfigHome(t *testing.T) {
	repoRoot := t.TempDir()
	xdgConfigHome := filepath.Join(repoRoot, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	t.Setenv("FORCE", "")
	runner := &autoloop.FakeRunner{Results: []autoloop.Result{{}, {}}}
	oldRunner := serviceRunner
	serviceRunner = runner
	t.Cleanup(func() {
		serviceRunner = oldRunner
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"service", "install"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	unitPath := filepath.Join(xdgConfigHome, "systemd", "user", "gormes-orchestrator.service")
	unit, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(unit), "WorkingDirectory="+repoRoot) {
		t.Fatalf("unit = %q, want workdir %q", unit, repoRoot)
	}

	wantCommands := []autoloop.Command{
		{Name: "systemctl", Args: []string{"--user", "daemon-reload"}},
		{Name: "systemctl", Args: []string{"--user", "enable", "--now", "gormes-orchestrator.service"}},
	}
	if !reflect.DeepEqual(runner.Commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", runner.Commands, wantCommands)
	}
}

func TestServiceInstallAuditUsesAuditUnitName(t *testing.T) {
	repoRoot := t.TempDir()
	xdgConfigHome := filepath.Join(repoRoot, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	runner := &autoloop.FakeRunner{Results: []autoloop.Result{{}, {}}}
	oldRunner := serviceRunner
	serviceRunner = runner
	t.Cleanup(func() {
		serviceRunner = oldRunner
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"service", "install-audit"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(xdgConfigHome, "systemd", "user", "gormes-orchestrator-audit.service")); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	wantEnable := autoloop.Command{Name: "systemctl", Args: []string{"--user", "enable", "--now", "gormes-orchestrator-audit.service"}}
	if got := runner.Commands[len(runner.Commands)-1]; !reflect.DeepEqual(got, wantEnable) {
		t.Fatalf("last command = %#v, want %#v", got, wantEnable)
	}
}

func TestServiceDisableLegacyTimersUsesRunner(t *testing.T) {
	runner := &autoloop.FakeRunner{Results: []autoloop.Result{{}, {}}}
	oldRunner := serviceRunner
	serviceRunner = runner
	t.Cleanup(func() {
		serviceRunner = oldRunner
	})

	if err := run([]string{"service", "disable", "legacy-timers"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	wantCommands := []autoloop.Command{
		{Name: "systemctl", Args: []string{"--user", "disable", "--now", "gormes-architecture-planner-tasks-manager.timer"}},
		{Name: "systemctl", Args: []string{"--user", "disable", "--now", "gormes-architectureplanneragent.timer"}},
	}
	if !reflect.DeepEqual(runner.Commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", runner.Commands, wantCommands)
	}
}

func TestServiceInstallUsesHomeWhenXDGConfigHomeEmpty(t *testing.T) {
	repoRoot := t.TempDir()
	home := filepath.Join(repoRoot, "home")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	runner := &autoloop.FakeRunner{Results: []autoloop.Result{{}, {}}}
	oldRunner := serviceRunner
	serviceRunner = runner
	t.Cleanup(func() {
		serviceRunner = oldRunner
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"service", "install"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".config", "systemd", "user", "gormes-orchestrator.service")); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestDigestIgnoresRunOnlyEnvValidation(t *testing.T) {
	repoRoot := t.TempDir()
	runRoot := filepath.Join(repoRoot, "custom-runs")
	ledgerPath := filepath.Join(runRoot, "state", "runs.jsonl")
	if err := autoloop.AppendLedgerEvent(ledgerPath, autoloop.LedgerEvent{
		TS:    time.Unix(1, 0).UTC(),
		Event: "run_started",
	}); err != nil {
		t.Fatalf("AppendLedgerEvent() error = %v", err)
	}
	t.Setenv("RUN_ROOT", runRoot)
	t.Setenv("MAX_AGENTS", "bad")

	var stdout bytes.Buffer
	oldStdout := commandStdout
	commandStdout = &stdout
	t.Cleanup(func() {
		commandStdout = oldStdout
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := run([]string{"digest"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "runs: 1") {
		t.Fatalf("stdout = %q, want digest from configured RUN_ROOT", stdout.String())
	}
}
