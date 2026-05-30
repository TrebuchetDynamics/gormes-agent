package configwriter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigWriter_EnvPathHonorsGormesHome(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	got := EnvPath()
	want := filepath.Join(gormesHome, ".env")
	if got != want {
		t.Fatalf("EnvPath() = %q, want %q", got, want)
	}
}

func TestConfigWriter_IsSecretKeyClassification(t *testing.T) {
	secret := []string{
		"api_key",
		"API_KEY",
		"OPENROUTER_API_KEY",
		"GITHUB_TOKEN",
		"telegram.bot_token",
	}
	notSecret := []string{
		"endpoint",
		"model",
		"hermes.endpoint",
		"hermes.model",
		"telegram.coalesce_ms",
		"goncho.enabled",
	}
	for _, k := range secret {
		if !IsSecretKey(k) {
			t.Errorf("IsSecretKey(%q) = false, want true", k)
		}
	}
	for _, k := range notSecret {
		if IsSecretKey(k) {
			t.Errorf("IsSecretKey(%q) = true, want false", k)
		}
	}
}

func TestConfigWriter_TelegramSecretAndHomeChannel(t *testing.T) {
	if got := SecretEnvName("telegram.bot_token"); got != "GORMES_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("SecretEnvName(telegram.bot_token) = %q, want GORMES_TELEGRAM_BOT_TOKEN", got)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := WriteTOMLValue(path, "telegram.home_channel.chat_id", "123456789012345678"); err != nil {
		t.Fatalf("WriteTOMLValue home_channel.chat_id: %v", err)
	}
	if err := WriteTOMLValue(path, "telegram.home_channel.thread_id", "42"); err != nil {
		t.Fatalf("WriteTOMLValue home_channel.thread_id: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "[telegram.home_channel]") {
		t.Fatalf("config.toml missing [telegram.home_channel]:\n%s", got)
	}
	if !strings.Contains(got, `chat_id = "123456789012345678"`) &&
		!strings.Contains(got, `chat_id = '123456789012345678'`) {
		t.Fatalf("config.toml missing string chat_id:\n%s", got)
	}
	if !strings.Contains(got, `thread_id = "42"`) &&
		!strings.Contains(got, `thread_id = '42'`) {
		t.Fatalf("config.toml missing string thread_id:\n%s", got)
	}
}

func TestConfigWriter_WriteTOMLValueSetsTopLevelHermesField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteTOMLValue(path, "endpoint", "https://example.invalid/v1"); err != nil {
		t.Fatalf("WriteTOMLValue endpoint: %v", err)
	}
	if err := WriteTOMLValue(path, "model", "test-model"); err != nil {
		t.Fatalf("WriteTOMLValue model: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "endpoint = 'https://example.invalid/v1'") &&
		!strings.Contains(got, `endpoint = "https://example.invalid/v1"`) {
		t.Fatalf("config.toml missing endpoint field:\n%s", got)
	}
	if !strings.Contains(got, "model = 'test-model'") &&
		!strings.Contains(got, `model = "test-model"`) {
		t.Fatalf("config.toml missing model field:\n%s", got)
	}
	if !strings.Contains(got, "[hermes]") {
		t.Fatalf("config.toml missing [hermes] section header:\n%s", got)
	}
}

func TestConfigWriter_WriteTOMLValueDottedSectionRoutesToTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteTOMLValue(path, "telegram.bot_username", "gormesbot"); err != nil {
		t.Fatalf("WriteTOMLValue: %v", err)
	}
	body, _ := os.ReadFile(path)
	got := string(body)
	if !strings.Contains(got, "[telegram]") {
		t.Fatalf("config.toml missing [telegram] section:\n%s", got)
	}
	if !strings.Contains(got, "bot_username = 'gormesbot'") &&
		!strings.Contains(got, `bot_username = "gormesbot"`) {
		t.Fatalf("config.toml missing telegram.bot_username:\n%s", got)
	}
}

func TestConfigWriter_WriteTOMLValuePreservesCommentsAndUnicode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`# operator-selected defaults
[hermes] # primary provider section
# keep the model rationale near the value
model = "old-model" # inline model note
endpoint = "https://example.invalid/v1"

[display]
skin = "default"
system_prompt = "你好，保持中文输出"
`), 0o600); err != nil {
		t.Fatalf("write seed config: %v", err)
	}

	if err := WriteTOMLValue(path, "hermes.model", "new-model"); err != nil {
		t.Fatalf("WriteTOMLValue: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(body)
	for _, want := range []string{
		"# operator-selected defaults",
		"[hermes] # primary provider section",
		"# keep the model rationale near the value",
		"# inline model note",
		"system_prompt = \"你好，保持中文输出\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config.toml lost %q after write:\n%s", want, got)
		}
	}
	if strings.Contains(got, "old-model") {
		t.Fatalf("config.toml kept old model after update:\n%s", got)
	}
	if !strings.Contains(got, `model = "new-model"`) && !strings.Contains(got, `model = 'new-model'`) {
		t.Fatalf("config.toml missing updated model:\n%s", got)
	}
	if strings.Contains(got, `\u4f60`) {
		t.Fatalf("config.toml escaped readable Unicode after unrelated write:\n%s", got)
	}
}

func TestConfigWriter_WriteTOMLValuePreservesSymlink(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real-config.toml")
	linkPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(realPath, []byte("[hermes]\nmodel = 'old'\n"), 0o644); err != nil {
		t.Fatalf("write real config: %v", err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := WriteTOMLValue(linkPath, "model", "new-model"); err != nil {
		t.Fatalf("WriteTOMLValue: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config link was replaced with mode %v, want symlink preserved", info.Mode())
	}
	got, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read real config: %v", err)
	}
	if !strings.Contains(string(got), "new-model") {
		t.Fatalf("real config was not updated through symlink:\n%s", got)
	}
}

func TestConfigWriter_WriteTOMLValueRejectsUnknownSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := WriteTOMLValue(path, "weatherman.endpoint", "x")
	if err == nil {
		t.Fatalf("WriteTOMLValue unknown section: err = nil, want typed error")
	}
	if !strings.Contains(err.Error(), "weatherman") {
		t.Fatalf("error %q does not name the offending section", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatalf("config.toml created on rejection, want no write")
	}
}

func TestConfigWriter_WriteTOMLValueAcceptsExplicitEmptyButRejectsMissingValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteTOMLValue(path, "endpoint", ""); err != nil {
		t.Fatalf("WriteTOMLValue empty: %v", err)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "endpoint = ''") &&
		!strings.Contains(string(body), `endpoint = ""`) {
		t.Fatalf("explicit empty endpoint not persisted:\n%s", string(body))
	}
}

func TestConfigWriter_WriteEnvValueAppendsAndUpdatesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	if err := WriteEnvValue(path, "GORMES_API_KEY", "sk-test"); err != nil {
		t.Fatalf("WriteEnvValue create: %v", err)
	}
	if err := WriteEnvValue(path, "OTHER_TOKEN", "abc"); err != nil {
		t.Fatalf("WriteEnvValue append: %v", err)
	}
	if err := WriteEnvValue(path, "GORMES_API_KEY", "sk-updated"); err != nil {
		t.Fatalf("WriteEnvValue update: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "GORMES_API_KEY=sk-updated\n") {
		t.Fatalf("GORMES_API_KEY not updated:\n%s", got)
	}
	if !strings.Contains(got, "OTHER_TOKEN=abc\n") {
		t.Fatalf("OTHER_TOKEN missing:\n%s", got)
	}
	count := strings.Count(got, "GORMES_API_KEY=")
	if count != 1 {
		t.Fatalf("GORMES_API_KEY occurrences = %d, want 1 (no duplicate lines)\nbody:\n%s", count, got)
	}
}

func TestConfigWriter_WriteEnvValueCreatesParentDirAndUsesRestrictivePerms(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "deeper", ".env")

	if err := WriteEnvValue(path, "GORMES_API_KEY", "sk-test"); err != nil {
		t.Fatalf("WriteEnvValue: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Fatalf("env perms = %o, want group/other unreadable", perm)
	}
}

func TestConfigWriter_WriteEnvValueTightensExistingDotenvPerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GORMES_API_KEY=sk-old\n"), 0o644); err != nil {
		t.Fatalf("seed .env: %v", err)
	}

	if err := WriteEnvValue(path, "GORMES_API_KEY", "sk-new"); err != nil {
		t.Fatalf("WriteEnvValue update: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("env perms = %o, want 600 after update", perm)
	}
}

func TestConfigWriterRoutesNavivoxTokenToEnv(t *testing.T) {
	if !IsSecretKey("navivox.token") {
		t.Fatal("navivox.token should be classified as a secret")
	}
	if got := SecretEnvName("navivox.token"); got != "GORMES_NAVIVOX_TOKEN" {
		t.Fatalf("SecretEnvName(navivox.token) = %q", got)
	}
}
