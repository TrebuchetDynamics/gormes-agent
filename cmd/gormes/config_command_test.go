package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestConfigCommand_PathAndEnvPathJSON proves
// `gormes config path --json` and `gormes config env-path --json`
// emit a parseable `{build, kind, path}` document so fleet automation
// inventorying Gormes config locations across machines can ingest each
// path with binary attribution. Build provenance leads — same
// convention as the rest of the `--json` arc. The default text output
// (single-line path) remains unchanged for shell-script consumers
// already using $(gormes config path) interpolation.
func TestConfigCommand_PathAndEnvPathJSON(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	for _, tc := range []struct {
		args     []string
		wantKind string
		wantPath string
	}{
		{[]string{"config", "path", "--json"}, "config", config.ConfigPath()},
		{[]string{"config", "env-path", "--json"}, "env", config.EnvPath()},
	} {
		t.Run(tc.wantKind, func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeOneshotFlagCommand(cmd, tc.args...)
			if err != nil {
				t.Fatalf("%v: %v\nstderr=%s", tc.args, err, stderr)
			}
			var got struct {
				Build struct {
					Version string `json:"version"`
				} `json:"build"`
				Kind string `json:"kind"`
				Path string `json:"path"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
				t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
			}
			if got.Build.Version != Version {
				t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
			}
			if got.Kind != tc.wantKind {
				t.Errorf("kind = %q, want %q", got.Kind, tc.wantKind)
			}
			if got.Path != tc.wantPath {
				t.Errorf("path = %q, want %q", got.Path, tc.wantPath)
			}
		})
	}
}

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

func TestConfigCommand_SetVoiceRecordKeyWritesTOMLAndLoads(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "voice.record_key", "ctrl+space")
	if err != nil {
		t.Fatalf("set voice.record_key: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "voice.record_key") || !strings.Contains(stdout, config.ConfigPath()) {
		t.Fatalf("stdout = %q, want voice.record_key and config path", stdout)
	}

	tomlBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	body := string(tomlBody)
	if !strings.Contains(body, "[voice]") ||
		(!strings.Contains(body, `record_key = 'ctrl+space'`) && !strings.Contains(body, `record_key = "ctrl+space"`)) {
		t.Fatalf("config.toml missing voice.record_key:\n%s", tomlBody)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load after voice.record_key write: %v", err)
	}
	if cfg.Voice.RecordKey != "ctrl+space" {
		t.Fatalf("loaded voice.record_key = %q, want ctrl+space", cfg.Voice.RecordKey)
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

// TestConfigCommand_ShowJSONEmitsStructuredRedactedDocument proves
// `gormes config show --json` returns a parseable
// `{build, paths: {config, env}, hermes: {endpoint, model, provider},
// secrets: {api_key, gormes_api_key_env}}` document with secrets
// reduced to `set` / `(not set)` markers — never raw values. Fleet
// dashboards use this to confirm config landed correctly across
// machines without scraping multi-section prose.
func TestConfigCommand_ShowJSONEmitsStructuredRedactedDocument(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
provider = "openai"
`))

	envDir := filepath.Join(filepath.Dir(config.ConfigPath()))
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	if err := os.WriteFile(config.EnvPath(),
		[]byte("GORMES_API_KEY=sk-not-leaked-1234\n"), 0o600); err != nil {
		t.Fatalf("seed .env: %v", err)
	}
	t.Setenv("GORMES_API_KEY", "sk-not-leaked-1234")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "show", "--json")
	if err != nil {
		t.Fatalf("config show --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Paths struct {
			Config string `json:"config"`
			Env    string `json:"env"`
		} `json:"paths"`
		Hermes struct {
			Endpoint string `json:"endpoint"`
			Model    string `json:"model"`
			Provider string `json:"provider"`
		} `json:"hermes"`
		Secrets struct {
			APIKey            string `json:"api_key"`
			GormesAPIKeyEnv   string `json:"gormes_api_key_env"`
		} `json:"secrets"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("config show --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Hermes.Endpoint != "https://example.invalid/v1" {
		t.Errorf("hermes.endpoint = %q, want example.invalid", got.Hermes.Endpoint)
	}
	if got.Hermes.Model != "test-model" {
		t.Errorf("hermes.model = %q, want test-model", got.Hermes.Model)
	}
	if got.Hermes.Provider != "openai" {
		t.Errorf("hermes.provider = %q, want openai", got.Hermes.Provider)
	}
	// Secrets MUST be redacted — never the raw token, even though the env
	// var is in scope.
	if strings.Contains(stdout, "sk-not-leaked-1234") {
		t.Fatalf("config show --json leaked raw secret:\n%s", stdout)
	}
	// And the secrets document MUST surface set/unset state so operators
	// can audit which credentials are configured per machine.
	if !strings.Contains(strings.ToLower(got.Secrets.GormesAPIKeyEnv), "set") {
		t.Errorf("secrets.gormes_api_key_env = %q, want a set/unset marker", got.Secrets.GormesAPIKeyEnv)
	}
}

// TestConfigCommand_CheckJSONEmitsStructuredReport proves
// `gormes config check --json` returns a parseable
// `{build, paths: {config, env}, config_version, latest_version,
// dotenv_present, issues: [{severity, field, message}], ok}` document
// so fleet automation can flag schema drift across machines without
// scraping bracketed prose. `ok` is the single boolean a CI pipeline
// can branch on.
func TestConfigCommand_CheckJSONEmitsStructuredReport(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
provider = "openai"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "check", "--json")
	if err != nil {
		t.Fatalf("config check --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Paths struct {
			Config string `json:"config"`
			Env    string `json:"env"`
		} `json:"paths"`
		ConfigVersion int  `json:"config_version"`
		LatestVersion int  `json:"latest_version"`
		DotenvPresent bool `json:"dotenv_present"`
		Issues        []struct {
			Severity string `json:"severity"`
			Field    string `json:"field"`
			Message  string `json:"message"`
		} `json:"issues"`
		OK bool `json:"ok"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("config check --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Paths.Config == "" {
		t.Errorf("paths.config must be populated")
	}
	if got.LatestVersion <= 0 {
		t.Errorf("latest_version = %d, want >0", got.LatestVersion)
	}
	if got.Issues == nil {
		t.Errorf("issues must be a JSON array (possibly empty), not null; got %+v", got)
	}
}

// TestConfigCommand_MigrateJSONEmitsStructuredOutcome proves
// `gormes config migrate --json` returns a parseable
// `{build, path, from_version, to_version, no_op, wrote}` document
// for fleet rollouts that need to confirm config.toml landed on the
// current schema version across machines. Operators driving rolling
// upgrades parse `wrote` to know which hosts actually applied the
// migration vs. were already current.
func TestConfigCommand_MigrateJSONEmitsStructuredOutcome(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	// Already-current config => no-op migration. Operators on freshly
	// installed machines hitting the migrate endpoint expect the JSON
	// to clearly say "no-op", so they don't think a write happened.
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://example.invalid/v1"
model = "test-model"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "migrate", "--json")
	if err != nil {
		t.Fatalf("config migrate --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Path        string `json:"path"`
		FromVersion int    `json:"from_version"`
		ToVersion   int    `json:"to_version"`
		NoOp        bool   `json:"no_op"`
		Wrote       bool   `json:"wrote"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("config migrate --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Path == "" {
		t.Errorf("path must be populated")
	}
	if got.ToVersion <= 0 {
		t.Errorf("to_version = %d, want >0", got.ToVersion)
	}
}

// TestConfigCommand_SetJSONEmitsStructuredOutcome proves
// `gormes config set <key> <value> --json` returns
// `{build, key, target, path, secret}` so fleet provisioning scripts
// can confirm WHERE the value landed (TOML vs dotenv) and whether it
// was treated as a secret. The raw value MUST never appear in JSON
// output — even non-secret values are excluded so the on-disk config
// remains the only source of truth and audit logs don't double-store
// configuration.
func TestConfigCommand_SetJSONEmitsStructuredOutcome(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "hermes.endpoint", "https://example.invalid/v1", "--json")
	if err != nil {
		t.Fatalf("config set --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Key    string `json:"key"`
		Target string `json:"target"`
		Path   string `json:"path"`
		Secret bool   `json:"secret"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("config set --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Key != "hermes.endpoint" {
		t.Errorf("key = %q, want %q", got.Key, "hermes.endpoint")
	}
	if got.Target != "toml" {
		t.Errorf("target = %q, want %q for non-secret key", got.Target, "toml")
	}
	if got.Secret {
		t.Errorf("secret = true for hermes.endpoint; should be false (non-secret key)")
	}
	if got.Path == "" {
		t.Errorf("path must point to the file that received the write")
	}
	// Raw VALUE must NEVER appear in JSON output — even non-secrets.
	if strings.Contains(stdout, "https://example.invalid/v1") {
		t.Fatalf("config set --json must not echo the raw value:\n%s", stdout)
	}
}

// TestConfigCommand_SetJSONSecretRoutesToDotenv proves a secret key
// (api_key) lands in the dotenv path with `secret: true` and `target:
// "dotenv"`. The raw secret value MUST NEVER appear in stdout.
func TestConfigCommand_SetJSONSecretRoutesToDotenv(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "config", "set", "api_key", "sk-must-not-leak-XYZ", "--json")
	if err != nil {
		t.Fatalf("config set api_key --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Key    string `json:"key"`
		Target string `json:"target"`
		Path   string `json:"path"`
		Secret bool   `json:"secret"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("config set --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if !got.Secret {
		t.Errorf("api_key must report secret=true; got %+v", got)
	}
	if got.Target != "dotenv" {
		t.Errorf("api_key target = %q, want %q", got.Target, "dotenv")
	}
	// Hard guarantee: the raw secret must not appear anywhere in stdout.
	if strings.Contains(stdout, "sk-must-not-leak-XYZ") {
		t.Fatalf("config set --json LEAKED the raw secret to stdout:\n%s", stdout)
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
