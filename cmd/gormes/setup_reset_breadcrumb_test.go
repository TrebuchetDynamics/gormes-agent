package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestSetupReset_PreservesBreadcrumbOfPriorConfig proves that
// `setup --reset` saves a recovery breadcrumb of the operator's
// previous config.toml as `config.toml.before-reset.<UTC>` BEFORE
// overwriting with defaults. Without this, an operator who hits
// `--reset` by accident loses everything (provider, endpoint, model,
// custom keys) and has no way to roll back. The breadcrumb is the
// minimum-effort safety net.
func TestSetupReset_PreservesBreadcrumbOfPriorConfig(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(dataHome, "gormes"))

	configPath := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	priorBody := "[hermes]\nprovider = 'openai'\nendpoint = 'https://example.com/v1'\nmodel = 'custom-prior'\n"
	if err := os.WriteFile(configPath, []byte(priorBody), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := resetSetupDefaultConfig(); err != nil {
		t.Fatalf("resetSetupDefaultConfig: %v", err)
	}

	// Find a breadcrumb whose name starts with "config.toml.before-reset."
	entries, err := os.ReadDir(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var breadcrumb string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "config.toml.before-reset.") {
			breadcrumb = filepath.Join(filepath.Dir(configPath), e.Name())
			break
		}
	}
	if breadcrumb == "" {
		t.Fatalf("expected a config.toml.before-reset.<ts> breadcrumb; dir contents: %v", dirNames(entries))
	}

	got, err := os.ReadFile(breadcrumb)
	if err != nil {
		t.Fatalf("read breadcrumb %s: %v", breadcrumb, err)
	}
	if string(got) != priorBody {
		t.Fatalf("breadcrumb body = %q, want exact prior config %q", got, priorBody)
	}
	// Mode must match the original 0o600 — the breadcrumb may carry
	// secrets (api_key, tokens) so it must not be world-readable.
	info, err := os.Stat(breadcrumb)
	if err != nil {
		t.Fatalf("stat breadcrumb: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("breadcrumb mode = %v, want 0o600 (must not leak secrets to other users)", info.Mode().Perm())
	}
}

// TestSetupReset_NoBreadcrumbWhenNoPriorConfig proves the reset path
// is a no-op for the breadcrumb when there's nothing to back up.
// Fresh installs that hit --reset before any config has been written
// shouldn't see a stray empty breadcrumb file.
func TestSetupReset_NoBreadcrumbWhenNoPriorConfig(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(dataHome, "gormes"))

	if _, err := resetSetupDefaultConfig(); err != nil {
		t.Fatalf("resetSetupDefaultConfig: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(config.ConfigPath()))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "config.toml.before-reset.") {
			t.Fatalf("fresh-install reset must NOT produce an empty breadcrumb; got %s", e.Name())
		}
	}
}

func dirNames(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name()
	}
	return out
}

// TestSetupReset_PrintsBreadcrumbPathToOperator proves the operator
// sees the breadcrumb path on stdout after `--reset`. Without this,
// the safety net is invisible — the file exists but the operator has
// no idea where to look to roll back. Surfacing the path is what
// turns "we wrote a backup somewhere" into a usable recovery handle.
func TestSetupReset_PrintsBreadcrumbPathToOperator(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte("[hermes]\nprovider = 'openai'\nmodel = 'custom-prior'\n"), 0o600); err != nil {
		t.Fatalf("write prior config: %v", err)
	}
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--reset", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "config.toml.before-reset.") {
		t.Fatalf("stdout must include the breadcrumb path so the operator can roll back; got:\n%s", stdout)
	}
}

// TestSetupReset_JSONEmitsStructuredOutcome proves
// `gormes setup --reset --json` returns
// `{build, action: "reset", config_path, breadcrumb_path}` so fleet
// automation running `setup --reset` across machines can audit where
// each operator's prior config was preserved without scraping the
// "Prior config preserved at X — restore with cp X Y" prose. The
// breadcrumb_path is the recovery handle scripts capture.
func TestSetupReset_JSONEmitsStructuredOutcome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte("[hermes]\nprovider = 'openai'\nmodel = 'fleet-snapshot'\n"), 0o600); err != nil {
		t.Fatalf("write prior config: %v", err)
	}
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--reset", "--non-interactive", "--json")
	if err != nil {
		t.Fatalf("setup --reset --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action         string `json:"action"`
		ConfigPath     string `json:"config_path"`
		BreadcrumbPath string `json:"breadcrumb_path"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("setup --reset --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "reset" {
		t.Errorf("action = %q, want %q", got.Action, "reset")
	}
	if got.ConfigPath != config.ConfigPath() {
		t.Errorf("config_path = %q, want %q", got.ConfigPath, config.ConfigPath())
	}
	if got.BreadcrumbPath == "" {
		t.Errorf("breadcrumb_path must be populated when prior config existed")
	}
	// Breadcrumb file MUST be on disk.
	if _, statErr := os.Stat(got.BreadcrumbPath); statErr != nil {
		t.Errorf("breadcrumb_path missing on disk: %v", statErr)
	}
}

// TestSetupReset_JSONFreshInstallEmptyBreadcrumb proves the JSON
// path keeps `breadcrumb_path: ""` (empty string, not absent) on a
// fresh install. Fleet scripts iterating across hosts get a stable
// shape regardless of whether the host had prior config.
func TestSetupReset_JSONFreshInstallEmptyBreadcrumb(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, _, err := runSetupTestCommand(t, fake.seams(), "--reset", "--non-interactive", "--json")
	if err != nil {
		t.Fatalf("setup --reset --json (fresh): %v", err)
	}
	var got struct {
		BreadcrumbPath string `json:"breadcrumb_path"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.BreadcrumbPath != "" {
		t.Errorf("fresh-install breadcrumb_path = %q, want empty string", got.BreadcrumbPath)
	}
}

// TestSetupReset_NoBreadcrumbLineWhenFreshInstall proves the reset
// path stays quiet about the breadcrumb when none was written. A
// fresh install hitting `--reset` should not see a phantom path that
// doesn't exist on disk.
func TestSetupReset_NoBreadcrumbLineWhenFreshInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--reset", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, "config.toml.before-reset.") {
		t.Fatalf("fresh-install reset must NOT print a breadcrumb path; got:\n%s", stdout)
	}
}
