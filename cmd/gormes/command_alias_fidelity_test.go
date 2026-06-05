package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

	t.Run("removed login command gives redacted auth add guidance", func(t *testing.T) {
		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "plain-secret-provider")
		if err == nil {
			t.Fatalf("login error = nil; stdout=%s stderr=%s", stdout, stderr)
		}
		combined := stdout + stderr + err.Error()
		for _, want := range []string{"unknown command \"login\"", "gormes auth add <provider> --type oauth"} {
			if !strings.Contains(combined, want) {
				t.Fatalf("login output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
			}
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

func setupMigrateOpenClawEnv(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("HOME", filepath.Join(root, "fake-home"))
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("GORMES_DISCORD_BOT_TOKEN", "")
	return root
}

func writeOpenClawCLIFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	cfg := `model: gpt-4.1-mini
providers:
  openrouter:
    api_key:
      source: env
      id: OPENROUTER_API_KEY
channels:
  telegram:
    bot_token:
      source: env
      id: TELEGRAM_BOT_TOKEN
mcp:
  servers:
    - name: notes
ui:
  theme: dark
unknown_top_level_section:
  ignored: true
`
	envBody := `TELEGRAM_BOT_TOKEN=plain-telegram-token
RANDOM_USER_VAR=plainvalue
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envBody), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# memory\n"), 0o600); err != nil {
		t.Fatalf("write memory: %v", err)
	}
}

func executeMigrateOpenClaw(args ...string) (*cobra.Command, string, string, error) {
	cmd := newMigrateCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return cmd, stdout.String(), stderr.String(), err
}
