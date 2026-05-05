package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestConfigCommand_PathSubcommandPrintsConfigPath(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "path")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	want := config.ConfigPath()
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestConfigCommand_EnvPathSubcommandPrintsDotenvPath(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "env-path")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	want := config.EnvPath()
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if !strings.HasSuffix(want, filepath.Join("gormes", ".env")) {
		t.Fatalf("env path %q does not point at gormes/.env", want)
	}
}

func TestConfigCommand_SetEndpointAndModelWritesTOMLOnly(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, _, err := executeOneshotFlagCommand(cmd, "config", "set", "endpoint", "https://example.invalid/v1"); err != nil {
		t.Fatalf("set endpoint: %v", err)
	}

	cmd2 := newRootCommandWithRuntime(rootRuntime{})
	if _, _, err := executeOneshotFlagCommand(cmd2, "config", "set", "model", "test-model"); err != nil {
		t.Fatalf("set model: %v", err)
	}

	tomlBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(tomlBody), "[hermes]") {
		t.Fatalf("config.toml missing [hermes]:\n%s", tomlBody)
	}
	if !strings.Contains(string(tomlBody), "https://example.invalid/v1") {
		t.Fatalf("config.toml missing endpoint value:\n%s", tomlBody)
	}
	if !strings.Contains(string(tomlBody), "test-model") {
		t.Fatalf("config.toml missing model value:\n%s", tomlBody)
	}
	if _, err := os.Stat(config.EnvPath()); !os.IsNotExist(err) {
		t.Fatalf(".env file exists after non-secret writes; stat err=%v", err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load after writes: %v", err)
	}
	if cfg.Hermes.Endpoint != "https://example.invalid/v1" {
		t.Fatalf("loaded endpoint = %q, want https://example.invalid/v1", cfg.Hermes.Endpoint)
	}
	if cfg.Hermes.Model != "test-model" {
		t.Fatalf("loaded model = %q, want test-model", cfg.Hermes.Model)
	}
}

func TestConfigCommand_SetTerminalCWDWritesTOMLAndLoads(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	projectDir := t.TempDir()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "terminal.cwd", projectDir)
	if err != nil {
		t.Fatalf("set terminal.cwd: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "terminal.cwd") || !strings.Contains(stdout, config.ConfigPath()) {
		t.Fatalf("stdout = %q, want terminal.cwd and config path", stdout)
	}

	tomlBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(tomlBody), "[terminal]") || !strings.Contains(string(tomlBody), projectDir) {
		t.Fatalf("config.toml missing terminal cwd:\n%s", tomlBody)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load after terminal.cwd write: %v", err)
	}
	if cfg.Terminal.CWD != projectDir {
		t.Fatalf("loaded terminal.cwd = %q, want %q", cfg.Terminal.CWD, projectDir)
	}
}

func TestConfigCommand_SetAPIKeyWritesEnvFileNotTOML(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, _, err := executeOneshotFlagCommand(cmd, "config", "set", "api_key", "sk-secret-123"); err != nil {
		t.Fatalf("set api_key: %v", err)
	}

	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_API_KEY=") {
		t.Fatalf(".env missing GORMES_API_KEY=:\n%s", envBody)
	}
	if !strings.Contains(string(envBody), "sk-secret-123") {
		t.Fatalf(".env missing api_key value:\n%s", envBody)
	}
	if data, err := os.ReadFile(config.ConfigPath()); err == nil {
		if strings.Contains(string(data), "sk-secret-123") {
			t.Fatalf("config.toml leaked secret into TOML:\n%s", data)
		}
		if strings.Contains(string(data), "api_key") {
			t.Fatalf("config.toml contains api_key key:\n%s", data)
		}
	}
}

func TestConfigCommand_SetMissingValueIsRejected(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "endpoint")
	if err == nil {
		t.Fatalf("Execute(missing value) err = nil; stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestConfigCommand_SetExplicitEmptyValueIsAccepted(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	if _, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "endpoint", ""); err != nil {
		t.Fatalf("Execute(explicit empty) err = %v stderr=%s", err, stderr)
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(body), "endpoint = ''") &&
		!strings.Contains(string(body), `endpoint = ""`) {
		t.Fatalf("explicit empty endpoint not persisted:\n%s", body)
	}
}

func TestConfigCommand_SetUnknownSectionIsRejected(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	_, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "weatherman.endpoint", "x")
	if err == nil {
		t.Fatalf("Execute(unknown section) err = nil stderr=%s", stderr)
	}
	if _, statErr := os.Stat(config.ConfigPath()); statErr == nil {
		t.Fatalf("config.toml was written for unknown section")
	}
}

func TestConfigCommand_ShowRedactsSecretsAndPrintsModelEndpoint(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`))

	envDir := filepath.Join(filepath.Dir(config.ConfigPath()))
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	if err := os.WriteFile(config.EnvPath(),
		[]byte("GORMES_API_KEY=sk-not-leaked-1234\n"), 0o600); err != nil {
		t.Fatalf("seed .env: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "show")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "https://example.invalid/v1") {
		t.Fatalf("show stdout missing endpoint:\n%s", stdout)
	}
	if !strings.Contains(stdout, "test-model") {
		t.Fatalf("show stdout missing model:\n%s", stdout)
	}
	if strings.Contains(stdout, "sk-not-leaked-1234") {
		t.Fatalf("show stdout leaked raw secret:\n%s", stdout)
	}
	if !strings.Contains(strings.ToLower(stdout), "set") {
		t.Fatalf("show stdout does not surface api_key set/unset state:\n%s", stdout)
	}
}
