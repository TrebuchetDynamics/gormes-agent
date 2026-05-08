package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestConfigGet_JSONNonSecretEmitsStructuredValue pins the contract
// for `gormes config get <key> --json` on non-secret keys: a parseable
// `{build, key, value, secret_redacted: false, set: bool}` document.
// Fleet automation that wants to scrape one config value
// programmatically previously had to pull the whole
// `config show --json` document and jq-traverse to the target key —
// this is the per-key shape consistent with `set`/`show`/`check`.
func TestConfigGet_JSONNonSecretEmitsStructuredValue(t *testing.T) {
	freshInstallE2EHome(t)

	// Seed a value via `config set` so the round-trip mirrors the
	// operator path.
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, _, err := executeRootCommandForTest(cmd, "config", "set", "hermes.model", "test-model-xyz"); err != nil {
		t.Fatalf("config set: %v", err)
	}

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "config", "get", "hermes.model", "--json")
	if err != nil {
		t.Fatalf("config get --json: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Key            string `json:"key"`
		Value          string `json:"value"`
		SecretRedacted bool   `json:"secret_redacted"`
		Set            bool   `json:"set"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Errorf("build.git_commit must be populated")
	}
	if got.Key != "hermes.model" {
		t.Errorf("key = %q, want %q", got.Key, "hermes.model")
	}
	if got.Value != "test-model-xyz" {
		t.Errorf("value = %q, want %q", got.Value, "test-model-xyz")
	}
	if got.SecretRedacted {
		t.Errorf("secret_redacted = true on non-secret key")
	}
	if !got.Set {
		t.Errorf("set = false after `config set`")
	}
}

// TestConfigGet_JSONSecretKeyStaysRedacted is the regression fence:
// secret keys (api_key, *_TOKEN, etc.) must NEVER leak the raw value
// through `--json` — the `value` field carries the same `(set)` /
// `(not set)` placeholder the text surface prints, and
// `secret_redacted: true` lets consumers branch.
//
// This test deliberately does NOT call freshInstallE2EHome: that
// helper t.Setenvs `GORMES_API_KEY=""` for safety, which marks the
// key as "shell-set" in dotenv's shell-precedence map and BLOCKS
// the post-set dotenv overlay we need to inspect. Use isolated
// per-test t.Setenv that doesn't pre-register GORMES_API_KEY.
func TestConfigGet_JSONSecretKeyStaysRedacted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldAPIKey, hadAPIKey := os.LookupEnv("GORMES_API_KEY")
	if err := os.Unsetenv("GORMES_API_KEY"); err != nil {
		t.Fatalf("unset GORMES_API_KEY: %v", err)
	}
	t.Cleanup(func() {
		if hadAPIKey {
			_ = os.Setenv("GORMES_API_KEY", oldAPIKey)
			return
		}
		_ = os.Unsetenv("GORMES_API_KEY")
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, _, err := executeRootCommandForTest(cmd, "config", "set", "hermes.api_key", "sk-LEAK-DETECTOR-do-not-show"); err != nil {
		t.Fatalf("config set api_key: %v", err)
	}

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "config", "get", "hermes.api_key", "--json")
	if err != nil {
		t.Fatalf("config get hermes.api_key --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout+stderr, "sk-LEAK-DETECTOR") {
		t.Fatalf("config get --json LEAKED secret value into output:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got struct {
		Key            string `json:"key"`
		Value          string `json:"value"`
		SecretRedacted bool   `json:"secret_redacted"`
		Set            bool   `json:"set"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if !got.SecretRedacted {
		t.Errorf("secret_redacted = false on hermes.api_key (a secret key)")
	}
	// Value must be a redaction marker, not the raw secret.
	if !strings.Contains(got.Value, "set") {
		t.Errorf("value = %q, want a redacted placeholder (e.g. `set [REDACTED]`)", got.Value)
	}
}
