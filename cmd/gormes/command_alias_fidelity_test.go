package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHermesCommandAliasFidelity_RootUnknownAndTypoSuggestions(t *testing.T) {
	t.Run("unknown top level command stays nonzero guidance", func(t *testing.T) {
		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeRootCommandForTest(cmd, "no-such-command-xyzzy")
		if err == nil {
			t.Fatalf("unknown command error = nil; stdout=%s stderr=%s", stdout, stderr)
		}
		if exitCodeFromError(err) == 0 {
			t.Fatalf("unknown command exit code = 0, want nonzero")
		}
		combined := stdout + stderr + err.Error()
		if !strings.Contains(strings.ToLower(combined), "unknown command") {
			t.Fatalf("unknown command output missing guidance:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
		}
	})

	t.Run("login unsupported provider is explicit redacted guidance", func(t *testing.T) {
		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "plain-secret-provider")
		if err == nil {
			t.Fatalf("login error = nil; stdout=%s stderr=%s", stdout, stderr)
		}
		combined := stdout + stderr + err.Error()
		if !strings.Contains(combined, "auth_login_provider_unsupported") || !strings.Contains(combined, "allowed=nous|openai-codex") {
			t.Fatalf("login output missing unsupported-provider guidance:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
		}
		if strings.Contains(combined, "plain-secret-provider") {
			t.Fatalf("login guidance leaked provider argument:\n%s", combined)
		}
	})

	t.Run("migrate ooenclaw remains typo guidance not alias", func(t *testing.T) {
		root := setupMigrateOpenClawEnv(t)
		src := root + "/src"
		writeOpenClawCLIFixture(t, src)

		_, stdout, stderr, err := executeMigrateOpenClaw("ooenclaw", "--dry-run", "--source", src)
		if err == nil {
			t.Fatalf("migrate ooenclaw error = nil; stdout=%s stderr=%s", stdout, stderr)
		}
		combined := stdout + stderr + err.Error()
		if !strings.Contains(combined, "openclaw") || !strings.Contains(strings.ToLower(combined), "unknown command") {
			t.Fatalf("migrate ooenclaw output missing explicit openclaw typo guidance:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
		}
	})
}

func TestHermesCommandAliasFidelity_ClawMigrateDryRunDelegatesToOpenClawMigration(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	src := root + "/src"
	writeOpenClawCLIFixture(t, src)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "claw", "migrate", "--dry-run", "--source", src)
	if err != nil {
		t.Fatalf("claw migrate --dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var doc struct {
		Source struct {
			Selected     string `json:"selected"`
			SelectedPath string `json:"selected_path"`
		} `json:"source"`
		Counts struct {
			Migrated int `json:"migrated"`
			Skipped  int `json:"skipped"`
			Archived int `json:"archived"`
			Errors   int `json:"errors"`
		} `json:"counts"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if doc.Source.Selected != "explicit_source" || doc.Source.SelectedPath != src {
		t.Fatalf("unexpected source: %+v", doc.Source)
	}
	if doc.Counts.Migrated < 1 || doc.Counts.Skipped < 1 || doc.Counts.Archived < 1 || doc.Counts.Errors != 0 {
		t.Fatalf("unexpected counts: %+v", doc.Counts)
	}
	if strings.Contains(stdout, "plain-telegram-token") {
		t.Fatalf("stdout leaked raw secret: %s", stdout)
	}
}
