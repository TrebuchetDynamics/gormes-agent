package gormescli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestSetupResetPrintsBreadcrumbPathToOperator(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte("[hermes]\nprovider = 'openai'\nmodel = 'custom-prior'\n"), 0o600); err != nil {
		t.Fatalf("write prior config: %v", err)
	}
	breadcrumb, err := ResetSetupDefaultConfig()
	if err != nil {
		t.Fatalf("ResetSetupDefaultConfig: %v", err)
	}
	var out bytes.Buffer
	done, err := EmitSetupResetResult(&out, VersionBuildProvenance{Version: "test", GitCommit: "abc"}, config.ConfigPath(), breadcrumb, false)
	if err != nil {
		t.Fatalf("EmitSetupResetResult: %v", err)
	}
	if done {
		t.Fatalf("text reset result done = true, want false so setup can continue")
	}
	if !strings.Contains(out.String(), "config.toml.before-reset.") {
		t.Fatalf("stdout must include the breadcrumb path so the operator can roll back; got:\n%s", out.String())
	}
}

func TestSetupResetJSONEmitsStructuredOutcome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte("[hermes]\nprovider = 'openai'\nmodel = 'fleet-snapshot'\n"), 0o600); err != nil {
		t.Fatalf("write prior config: %v", err)
	}
	breadcrumb, err := ResetSetupDefaultConfig()
	if err != nil {
		t.Fatalf("ResetSetupDefaultConfig: %v", err)
	}
	var out bytes.Buffer
	done, err := EmitSetupResetResult(&out, VersionBuildProvenance{Version: "test-version", GitCommit: "abc"}, config.ConfigPath(), breadcrumb, true)
	if err != nil {
		t.Fatalf("EmitSetupResetResult: %v", err)
	}
	if !done {
		t.Fatalf("json reset result done = false, want true")
	}

	var got SetupResetReportJSON
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("setup reset JSON must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.Build.Version != "test-version" {
		t.Errorf("got.build.version = %q", got.Build.Version)
	}
	if got.Action != "reset" {
		t.Errorf("action = %q, want reset", got.Action)
	}
	if got.ConfigPath != config.ConfigPath() {
		t.Errorf("config_path = %q, want %q", got.ConfigPath, config.ConfigPath())
	}
	if got.BreadcrumbPath == "" {
		t.Errorf("breadcrumb_path must be populated when prior config existed")
	}
	if _, statErr := os.Stat(got.BreadcrumbPath); statErr != nil {
		t.Errorf("breadcrumb_path missing on disk: %v", statErr)
	}
}

func TestSetupResetJSONFreshInstallEmptyBreadcrumb(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	breadcrumb, err := ResetSetupDefaultConfig()
	if err != nil {
		t.Fatalf("ResetSetupDefaultConfig: %v", err)
	}
	var out bytes.Buffer
	_, err = EmitSetupResetResult(&out, VersionBuildProvenance{Version: "test"}, config.ConfigPath(), breadcrumb, true)
	if err != nil {
		t.Fatalf("EmitSetupResetResult: %v", err)
	}
	var got SetupResetReportJSON
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.BreadcrumbPath != "" {
		t.Errorf("fresh-install breadcrumb_path = %q, want empty string", got.BreadcrumbPath)
	}
}

func TestSetupResetNoBreadcrumbLineWhenFreshInstall(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	breadcrumb, err := ResetSetupDefaultConfig()
	if err != nil {
		t.Fatalf("ResetSetupDefaultConfig: %v", err)
	}
	var out bytes.Buffer
	_, err = EmitSetupResetResult(&out, VersionBuildProvenance{Version: "test"}, config.ConfigPath(), breadcrumb, false)
	if err != nil {
		t.Fatalf("EmitSetupResetResult: %v", err)
	}
	if strings.Contains(out.String(), "config.toml.before-reset.") {
		t.Fatalf("fresh-install reset must NOT print a breadcrumb path; got:\n%s", out.String())
	}
}
