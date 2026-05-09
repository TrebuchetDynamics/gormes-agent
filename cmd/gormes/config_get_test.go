package main

import (
	"strings"
	"testing"
)

// TestConfigGet_NonSecretReturnsValue: `gormes config get` is the
// natural counterpart to `gormes config set` — without it the CLI is
// asymmetric (`set` works, no `get` to read a single value back).
// Operators currently have to grep through `gormes config show` text
// or read config.toml directly. This test pins the contract for
// non-secret keys: the value printed verbatim, exit 0.
func TestConfigGet_NonSecretReturnsValue(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	// Seed a value via the existing `set` command so the test mirrors
	// the operator's intended round-trip path.
	if _, _, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "set", "hermes.model", "gpt-test-model",
	); err != nil {
		t.Fatalf("config set: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "get", "hermes.model",
	)
	if err != nil {
		t.Fatalf("config get: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != "gpt-test-model" {
		t.Errorf("config get hermes.model stdout = %q, want \"gpt-test-model\"", stdout)
	}
}

// TestConfigGet_SecretKeyEmitsRedactedStatus: secret keys must NOT
// print the raw value (same redaction promise as `config show`).
// Instead emit "(set)" or "(not set)" so operators can verify
// presence without leaking credentials into shell history.
func TestConfigGet_SecretKeyEmitsRedactedStatus(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	// `config set` writes the secret to .env and triggers
	// internal/config/dotenv.go to overlay GORMES_API_KEY into the
	// process env via os.Setenv. Track it on t so the test runner
	// cleanup restores it to whatever it was before — without this,
	// downstream tests (e.g. TestOnboardWizardInteractivePromptsForStepActions)
	// inherit a leaked secret and report different state.
	t.Setenv("GORMES_API_KEY", "")

	// Seed a secret. `config set` routes secret keys to .env.
	if _, _, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "set", "hermes.api_key", "sk-VERY-SECRET-VALUE-do-not-leak",
	); err != nil {
		t.Fatalf("config set api_key: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "get", "hermes.api_key",
	)
	if err != nil {
		t.Fatalf("config get api_key: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	combined := stdout + stderr
	if strings.Contains(combined, "sk-VERY-SECRET") {
		t.Fatalf("config get must redact secret values; leaked:\n%s", combined)
	}
	if !strings.Contains(combined, "(set)") && !strings.Contains(combined, "set") {
		t.Errorf("config get api_key must indicate the secret is set without leaking; got:\n%s", combined)
	}
}

// TestConfigGet_UnknownKeyErrors: an unknown config key must return
// a non-zero exit with a clear error so operators don't silently
// receive empty output (which is indistinguishable from "value is
// empty string").
func TestConfigGet_UnknownKeyErrors(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "get", "hermes.notarealkey",
	)
	if err == nil {
		t.Fatalf("unknown key must error; stdout=%q stderr=%q", stdout, stderr)
	}
	combined := err.Error() + stderr
	if !strings.Contains(combined, "notarealkey") {
		t.Errorf("error must name the rejected key; got:\n%s", combined)
	}
}

// TestConfigGet_NoArgErrors: zero args is a usage error, not a
// silent default. Mirrors `config set` which also rejects missing
// args.
func TestConfigGet_NoArgErrors(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	_, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "get",
	)
	if err == nil {
		t.Fatalf("config get with no key must error; stderr=%q", stderr)
	}
}

// TestConfigGet_EmptyNonSecretEmitsNotSetMarker pins UX consistency:
// when a non-secret key resolves to the empty string, emit `(not set)`
// rather than a blank line. Without this, `gormes config get
// hermes.endpoint` on a fresh install prints a single empty line that
// looks like a hung command, while `gormes config get hermes.api_key`
// (secret) already prints `(not set)`. The two should agree.
func TestConfigGet_EmptyNonSecretEmitsNotSetMarker(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"config", "get", "hermes.endpoint",
	)
	if err != nil {
		t.Fatalf("config get hermes.endpoint: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("config get on empty key must emit a placeholder (e.g. \"(not set)\") not a silent blank line; got stdout=%q", stdout)
	}
	if !strings.Contains(stdout, "not set") {
		t.Errorf("placeholder must read like \"(not set)\" to match the secret-key form; got %q", stdout)
	}
}
