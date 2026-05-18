package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestConfigEdit_NoEditorReportsConfigPath: when EDITOR/VISUAL is unset and
// no fallback editor binary is discoverable, `gormes config edit` must NOT
// launch a real shell. It must print a "no editor" message that includes
// the resolved config path so operators know where to point their editor.
func TestConfigEdit_NoEditorReportsConfigPath(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	withConfigEditorRunner(t, stubEditorRunner{
		lookPath: func(string) (string, bool) { return "", false },
		run:      func(string, string) error { t.Fatal("editor must not run when none is available"); return nil },
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "edit")
	if err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, config.ConfigPath()) {
		t.Fatalf("edit stdout missing config path:\n%s", stdout)
	}
	if !strings.Contains(strings.ToLower(stdout), "no editor") {
		t.Fatalf("edit stdout missing no-editor evidence:\n%s", stdout)
	}
}

// TestConfigEdit_CreatesConfigFileBeforeOpening: when config.toml does not
// exist, edit must materialize the file with at least the current schema
// config_version stamped, then dispatch the injected runner against the
// resolved EDITOR.
func TestConfigEdit_CreatesConfigFileBeforeOpening(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("EDITOR", "fake-editor")

	var ranEditor, ranPath string
	withConfigEditorRunner(t, stubEditorRunner{
		lookPath: func(name string) (string, bool) { return "/usr/bin/" + name, true },
		run: func(editor, path string) error {
			ranEditor, ranPath = editor, path
			return nil
		},
	})
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, stderr, err := executeOneshotFlagCommand(cmd, "config", "edit"); err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if ranEditor != "fake-editor" {
		t.Fatalf("editor = %q, want fake-editor", ranEditor)
	}
	if ranPath != config.ConfigPath() {
		t.Fatalf("editor path = %q, want %q", ranPath, config.ConfigPath())
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	if !strings.Contains(string(body), "config_version = 2") || !strings.Contains(string(body), "[profiles.main]") {
		t.Fatalf("created config.toml missing v2 seed:\n%s", body)
	}
}

// TestConfigEdit_PrefersEditorOverVisualAndFallback: precedence is
// EDITOR > VISUAL > common-editor lookup. The injected runner must be the
// only path through which a binary is invoked.
func TestConfigEdit_PrefersEditorOverVisualAndFallback(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("EDITOR", "explicit-editor")
	t.Setenv("VISUAL", "visual-editor")

	var got string
	withConfigEditorRunner(t, stubEditorRunner{
		lookPath: func(name string) (string, bool) { return "/usr/bin/" + name, true },
		run:      func(editor, _ string) error { got = editor; return nil },
	})
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, stderr, err := executeOneshotFlagCommand(cmd, "config", "edit"); err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if got != "explicit-editor" {
		t.Fatalf("EDITOR not preferred; got %q", got)
	}
}

// TestConfigEdit_FallsBackToCommonEditorWhenEnvUnset: when EDITOR/VISUAL are
// empty but a common editor (nano/vim/vi) is on PATH, edit must dispatch the
// first one found.
func TestConfigEdit_FallsBackToCommonEditorWhenEnvUnset(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	var got string
	withConfigEditorRunner(t, stubEditorRunner{
		lookPath: func(name string) (string, bool) {
			if name == "vim" {
				return "/usr/bin/vim", true
			}
			return "", false
		},
		run: func(editor, _ string) error { got = editor; return nil },
	})
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, stderr, err := executeOneshotFlagCommand(cmd, "config", "edit"); err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if got != "vim" {
		t.Fatalf("fallback editor = %q, want vim", got)
	}
}

// TestConfigCheck_ReportsConfigVersionAndDotenvAvailability: native check
// must surface config_version and report whether a dotenv file is present
// at the resolved env path. It must not write anything.
func TestConfigCheck_ReportsConfigVersionAndDotenvAvailability(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
config_version = 2

[profiles.main]
enabled = true
name = ""

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
provider = "openai"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "check")
	if err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "config_version") || !strings.Contains(stdout, "2") {
		t.Fatalf("check stdout missing config_version=2:\n%s", stdout)
	}
	if !strings.Contains(strings.ToLower(stdout), "dotenv") {
		t.Fatalf("check stdout missing dotenv evidence:\n%s", stdout)
	}
	if _, statErr := os.Stat(config.EnvPath()); statErr == nil {
		t.Fatalf("check unexpectedly created dotenv file at %s", config.EnvPath())
	}
}

// TestConfigCheck_ReportsMissingProviderFields: when the endpoint or model
// is empty, check must emit a missing-field issue. Empty strings on
// hermes.endpoint count as configured-but-empty and are flagged.
func TestConfigCheck_ReportsMissingProviderFields(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
config_version = 2

[hermes]
endpoint = ""
model = ""
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeOneshotFlagCommand(cmd, "config", "check")
	if err == nil {
		t.Fatalf("Execute(missing fields) err = nil; want non-nil for invalid config")
	}
	lower := strings.ToLower(stdout)
	if !strings.Contains(lower, "endpoint") || !strings.Contains(lower, "model") {
		t.Fatalf("check stdout missing endpoint/model evidence:\n%s", stdout)
	}
	if !strings.Contains(lower, "configured-but-empty") {
		t.Fatalf("check stdout did not mark explicit-empty values:\n%s", stdout)
	}
}

// TestConfigCheck_FutureVersionFails: a config.toml stamped with a
// config_version greater than CurrentConfigVersion must be reported as a
// future-version error and exit non-zero.
func TestConfigCheck_FutureVersionFails(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
config_version = 99

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeOneshotFlagCommand(cmd, "config", "check")
	if err == nil {
		t.Fatalf("Execute(future version) err = nil; want non-nil")
	}
	lower := strings.ToLower(stdout)
	if !strings.Contains(lower, "newer binary") && !strings.Contains(lower, "future") {
		t.Fatalf("check stdout missing future-version evidence:\n%s", stdout)
	}
}

// TestConfigCheck_RedactsSecrets: dotenv values and api_key TOML values must
// be redacted in check output.
func TestConfigCheck_RedactsSecrets(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
config_version = 2

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
api_key = "sk-leaky-toml-1234"
`))
	if err := os.MkdirAll(filepath.Dir(config.EnvPath()), 0o700); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	if err := os.WriteFile(config.EnvPath(), []byte("GORMES_API_KEY=sk-leaky-env-5678\n"), 0o600); err != nil {
		t.Fatalf("seed .env: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, _ := executeOneshotFlagCommand(cmd, "config", "check")
	if strings.Contains(stdout, "sk-leaky-toml-1234") {
		t.Fatalf("check leaked TOML api_key value:\n%s", stdout)
	}
	if strings.Contains(stdout, "sk-leaky-env-5678") {
		t.Fatalf("check leaked dotenv value:\n%s", stdout)
	}
}

// TestConfigMigrate_NoOpWhenCurrent: with a config already at
// CurrentConfigVersion, migrate must report no-op and leave the file's
// bytes unchanged.
func TestConfigMigrate_NoOpWhenCurrent(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	body := []byte(`config_version = 2

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"

[profiles.main]
enabled = true
name = ""
`)
	writeOneshotFlagConfig(t, body)
	before, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read pre: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "migrate")
	if err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(strings.ToLower(stdout), "no-op") && !strings.Contains(strings.ToLower(stdout), "already") {
		t.Fatalf("migrate stdout missing no-op evidence:\n%s", stdout)
	}
	after, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read post: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("migrate mutated config.toml on no-op:\nbefore=%s\nafter=%s", before, after)
	}
}

// TestConfigMigrate_StampsVersionOnUnversionedConfig: a pre-version-1 file
// (no config_version key) must be migrated to CurrentConfigVersion via
// atomic write — never a partial write.
func TestConfigMigrate_StampsVersionOnUnversionedConfig(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "migrate")
	if err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(strings.ToLower(stdout), "migrated") && !strings.Contains(strings.ToLower(stdout), "wrote") {
		t.Fatalf("migrate stdout missing migration evidence:\n%s", stdout)
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read post: %v", err)
	}
	if !strings.Contains(string(body), "config_version = 2") || !strings.Contains(string(body), "[profiles.main]") {
		t.Fatalf("migrated config.toml missing v2 seed:\n%s", body)
	}
	// No stale temp file from atomic rename.
	dir := filepath.Dir(config.ConfigPath())
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == "config.toml" || name == ".env" {
			continue
		}
		t.Fatalf("unexpected leftover file in %s: %s", dir, name)
	}
}

// TestConfigMigrate_RejectsFutureVersion: a config stamped with a future
// version must be rejected without rewriting the file.
func TestConfigMigrate_RejectsFutureVersion(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	body := []byte(`config_version = 99

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`)
	writeOneshotFlagConfig(t, body)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	_, _, err := executeOneshotFlagCommand(cmd, "config", "migrate")
	if err == nil {
		t.Fatalf("Execute(future version) err = nil; want non-nil")
	}
	after, readErr := os.ReadFile(config.ConfigPath())
	if readErr != nil {
		t.Fatalf("read post: %v", readErr)
	}
	if string(after) != string(body) {
		t.Fatalf("migrate rewrote future-version config:\n%s", after)
	}
}

// TestConfigCloseout_MigrateHelpDistinguishesNativeFromCrossProduct: the
// help text for `gormes config migrate` must clarify native schema scope and
// must not promise importing Hermes/OpenClaw state.
func TestConfigCloseout_MigrateHelpDistinguishesNativeFromCrossProduct(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeOneshotFlagCommand(cmd, "config", "migrate", "--help")
	if err != nil {
		t.Fatalf("Execute --help: %v", err)
	}
	lower := strings.ToLower(stdout)
	if !strings.Contains(lower, "native") {
		t.Fatalf("migrate --help missing 'native' qualifier:\n%s", stdout)
	}
	for _, banned := range []string{"~/.hermes", "~/.openclaw"} {
		if strings.Contains(stdout, banned) {
			t.Fatalf("migrate --help references cross-product path %q:\n%s", banned, stdout)
		}
	}
}

// stubEditorRunner is the test double for the injected editor lookup +
// runner. Production code uses os/exec; tests must never spawn one.
type stubEditorRunner struct {
	lookPath func(string) (string, bool)
	run      func(editor, path string) error
}

func (s stubEditorRunner) LookPath(name string) (string, bool) { return s.lookPath(name) }
func (s stubEditorRunner) Run(editor, path string) error       { return s.run(editor, path) }

var _ editorRunner = stubEditorRunner{}
