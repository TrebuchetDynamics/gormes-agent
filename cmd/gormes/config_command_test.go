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

func TestConfigCommand_SetTelegramBotTokenWritesLoadableEnvName(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	unsetEnvForConfigCommandTest(t, "GORMES_TELEGRAM_TOKEN", "TELEGRAM_BOT_TOKEN", "TELEGRAM_TOKEN")

	secret := "123456:telegram-secret"
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "telegram.bot_token", secret)
	if err != nil {
		t.Fatalf("set telegram.bot_token: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "GORMES_TELEGRAM_TOKEN") || strings.Contains(stdout, secret) {
		t.Fatalf("stdout = %q, want env name only and no raw secret", stdout)
	}

	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_TELEGRAM_TOKEN="+secret) {
		t.Fatalf(".env missing loadable Telegram token env name:\n%s", envBody)
	}
	if strings.Contains(string(envBody), "TELEGRAM.BOT_TOKEN=") {
		t.Fatalf(".env used dotted env name that config.Load ignores:\n%s", envBody)
	}
	if data, err := os.ReadFile(config.ConfigPath()); err == nil && strings.Contains(string(data), secret) {
		t.Fatalf("config.toml leaked Telegram token:\n%s", data)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load after telegram.bot_token write: %v", err)
	}
	if cfg.Telegram.BotToken != secret {
		t.Fatalf("loaded Telegram token = %q, want configured secret", cfg.Telegram.BotToken)
	}
}

func TestConfigCommand_SetTelegramAllowedUserIDsWritesLoadableList(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "telegram.allowed_user_ids", "6586915095")
	if err != nil {
		t.Fatalf("set telegram.allowed_user_ids: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "telegram.allowed_user_ids") || !strings.Contains(stdout, config.ConfigPath()) {
		t.Fatalf("stdout = %q, want key and config path", stdout)
	}

	data, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(data), "allowed_user_ids = [6586915095]") {
		t.Fatalf("config.toml did not write allowed_user_ids as TOML list:\n%s", data)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load after telegram.allowed_user_ids write: %v", err)
	}
	if got := cfg.Telegram.AllowedUserIDs; len(got) != 1 || got[0] != 6586915095 {
		t.Fatalf("AllowedUserIDs = %v, want [6586915095]", got)
	}
}

func TestConfigCommand_SetNestedAgentsDefaultsWorkspaceWritesTOMLAndLoads(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	workspace := filepath.Join(t.TempDir(), "workspace-gormes")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "agents.defaults.workspace", workspace)
	if err != nil {
		t.Fatalf("set agents.defaults.workspace: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "agents.defaults.workspace") || !strings.Contains(stdout, config.ConfigPath()) {
		t.Fatalf("stdout = %q, want nested key and config path", stdout)
	}

	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "[agents.defaults]") || !strings.Contains(got, workspace) {
		t.Fatalf("config.toml missing nested agents defaults workspace:\n%s", got)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load after agents defaults write: %v", err)
	}
	if len(cfg.Agents.List) == 0 || cfg.Agents.List[0].Workspace != workspace {
		t.Fatalf("default agent workspace = %+v, want %q", cfg.Agents.List, workspace)
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

func unsetEnvForConfigCommandTest(t *testing.T, keys ...string) {
	t.Helper()
	type savedEnv struct {
		key   string
		value string
		set   bool
	}
	saved := make([]savedEnv, 0, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		saved = append(saved, savedEnv{key: key, value: value, set: ok})
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		for _, item := range saved {
			if item.set {
				_ = os.Setenv(item.key, item.value)
			} else {
				_ = os.Unsetenv(item.key)
			}
		}
	})
}
