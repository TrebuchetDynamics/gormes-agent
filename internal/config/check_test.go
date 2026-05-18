package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigCheck_ReportsCurrentVersionWhenComplete: a fully-formed config
// at CurrentConfigVersion with non-empty endpoint+model returns no issues
// and reports both the resolved version and the dotenv presence flag.
func TestConfigCheck_ReportsCurrentVersionWhenComplete(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "config", "gormes"))

	configDir := filepath.Join(root, "config", "gormes")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(`config_version = 2

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`)
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), body, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	report, err := Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if report.ConfigVersion != CurrentConfigVersion {
		t.Fatalf("ConfigVersion = %d, want %d", report.ConfigVersion, CurrentConfigVersion)
	}
	if report.LatestVersion != CurrentConfigVersion {
		t.Fatalf("LatestVersion = %d, want %d", report.LatestVersion, CurrentConfigVersion)
	}
	if report.DotenvPresent {
		t.Fatalf("DotenvPresent = true; expected false (no .env seeded)")
	}
	if len(report.Issues) != 0 {
		t.Fatalf("Issues = %+v, want empty", report.Issues)
	}
}

// TestConfigCheck_FlagsExplicitEmptyAsConfiguredButEmpty: explicit empty
// strings on hermes.endpoint or hermes.model must surface as a
// configured-but-empty issue, distinct from missing-field.
func TestConfigCheck_FlagsExplicitEmptyAsConfiguredButEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "config", "gormes"))

	configDir := filepath.Join(root, "config", "gormes")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(`_config_version = 1

[hermes]
endpoint = ""
model = ""
`)
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), body, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	report, _ := Check()
	if len(report.Issues) == 0 {
		t.Fatalf("Issues = 0; want at least one configured-but-empty issue")
	}
	var sawEndpoint, sawModel bool
	for _, issue := range report.Issues {
		if issue.Severity != "error" {
			continue
		}
		if strings.Contains(issue.Field, "endpoint") && strings.Contains(strings.ToLower(issue.Message), "configured-but-empty") {
			sawEndpoint = true
		}
		if strings.Contains(issue.Field, "model") && strings.Contains(strings.ToLower(issue.Message), "configured-but-empty") {
			sawModel = true
		}
	}
	if !sawEndpoint || !sawModel {
		t.Fatalf("missing configured-but-empty issues for endpoint/model: %+v", report.Issues)
	}
}

// TestConfigCheck_FutureVersionReturnsError: when _config_version exceeds
// CurrentConfigVersion, Check returns a non-nil error and reports the field.
func TestConfigCheck_FutureVersionReturnsError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "config", "gormes"))

	configDir := filepath.Join(root, "config", "gormes")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(`_config_version = 99

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`)
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), body, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	report, err := Check()
	if err == nil {
		t.Fatalf("Check err = nil; want future-version error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "newer binary") {
		t.Fatalf("Check err = %v, want newer-binary message", err)
	}
	if report.ConfigVersion != 99 {
		t.Fatalf("ConfigVersion = %d, want 99 (raw)", report.ConfigVersion)
	}
}

// TestConfigCheck_DotenvPresenceFlagIsTrueWhenFileExists: when an .env file
// exists at EnvPath, DotenvPresent is true. Check never opens that file's
// contents — operators rely on `gormes config show` for redacted values.
func TestConfigCheck_DotenvPresenceFlagIsTrueWhenFileExists(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "config", "gormes"))

	configDir := filepath.Join(root, "config", "gormes")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`_config_version = 1

[hermes]
endpoint = "https://example.invalid/v1"
model = "m"
`), 0o600); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte("GORMES_API_KEY=sk-x\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	report, err := Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !report.DotenvPresent {
		t.Fatalf("DotenvPresent = false; want true")
	}
}

// TestConfigCheck_DoesNotMutateAnyFile: Check is read-only; every byte of
// the config and dotenv file must be byte-identical after the call.
func TestConfigCheck_DoesNotMutateAnyFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "config", "gormes"))

	configDir := filepath.Join(root, "config", "gormes")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tomlBody := []byte(`_config_version = 1

[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`)
	envBody := []byte("GORMES_API_KEY=sk-untouched\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), tomlBody, 0o600); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".env"), envBody, 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	if _, err := Check(); err != nil {
		t.Fatalf("Check: %v", err)
	}

	gotToml, _ := os.ReadFile(filepath.Join(configDir, "config.toml"))
	gotEnv, _ := os.ReadFile(filepath.Join(configDir, ".env"))
	if string(gotToml) != string(tomlBody) {
		t.Fatalf("config.toml mutated by Check:\n%s", gotToml)
	}
	if string(gotEnv) != string(envBody) {
		t.Fatalf(".env mutated by Check:\n%s", gotEnv)
	}
}
