package openclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixtureFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func writeOpenClawFixture(t *testing.T, root string) {
	t.Helper()
	writeFixtureFile(t, filepath.Join(root, "config.yaml"), `model: gpt-4.1-mini
custom_providers:
  - name: openrouter-custom
    base_url: https://openrouter.ai/api/v1
providers:
  openrouter:
    api_key:
      source: env
      id: OPENROUTER_API_KEY
channels:
  telegram:
    bot_token:
      source: env
      id: TELEGRAM_BOT_TOKEN
  slack:
    bot_token:
      source: file
      path: /etc/secrets/slack
mcp:
  servers:
    - name: notes
tts:
  provider: openai
approvals:
  mode: auto
tools:
  exec:
    timeout: 180
session_reset:
  daily: true
memory:
  enabled: true
ui:
  theme: dark
unknown_top_level_section:
  ignored: true
`)
	writeFixtureFile(t, filepath.Join(root, ".env"), `TELEGRAM_BOT_TOKEN=plain-telegram-token
DISCORD_BOT_TOKEN=plain-discord-token
OPENROUTER_API_KEY=plain-openrouter-key
RANDOM_USER_VAR=plainvalue
`)
	writeFixtureFile(t, filepath.Join(root, "MEMORY.md"), "# memory\n")
	writeFixtureFile(t, filepath.Join(root, "USER.md"), "# user\n")
	writeFixtureFile(t, filepath.Join(root, "skills", "demo", "SKILL.md"), "skill body\n")
}

func TestOpenClawManifestClassifiesConfigEnvFilesAndSecrets(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeOpenClawFixture(t, src)
	t.Setenv("HOME", filepath.Join(root, "fake-home"))

	m, err := BuildManifest(Options{Source: src, ExistingGormesEnv: map[string]string{"GORMES_DISCORD_BOT_TOKEN": "already-set"}})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	if m.Source.Selected != OriginExplicitSource || m.Source.SelectedPath != src {
		t.Fatalf("unexpected source: %+v", m.Source)
	}
	if got := findConfig(m, "model"); got == nil || got.Disposition != "importable" || got.GormesPath != "hermes.model" {
		t.Fatalf("model not importable: %+v", got)
	}
	if got := findConfig(m, "ui"); got == nil || got.Disposition != "archived" {
		t.Fatalf("ui not archived: %+v", got)
	}
	if got := findEnv(m, "DISCORD_BOT_TOKEN"); got == nil || got.Disposition != "conflict" || !got.ConflictWithExisting {
		t.Fatalf("discord env not conflict: %+v", got)
	}
	if got := findEnv(m, "TELEGRAM_BOT_TOKEN"); got == nil || got.Disposition != "importable" || got.RedactedValue != RedactedValue {
		t.Fatalf("telegram env not importable/redacted: %+v", got)
	}
	if got := findSecretRef(m, "channels.slack.bot_token"); got == nil || got.Disposition != "manual" || got.Source != "file" {
		t.Fatalf("file secret ref not manual: %+v", got)
	}
	for _, kind := range []string{"memory", "user_profile", "skills"} {
		if got := findFile(m, kind); got == nil || got.Disposition != "importable" {
			t.Fatalf("workspace file %s not importable: %+v", kind, got)
		}
	}
	raw, _ := json.Marshal(m)
	if strings.Contains(string(raw), "plain-telegram-token") || strings.Contains(string(raw), "plain-discord-token") || strings.Contains(string(raw), "plain-openrouter-key") {
		t.Fatalf("manifest leaked raw secret material: %s", raw)
	}
}

func TestOpenClawManifestFallbackAndMissingSource(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	dotOpenClaw := filepath.Join(home, ".openclaw")
	writeOpenClawFixture(t, dotOpenClaw)
	t.Setenv("HOME", home)

	m, err := BuildManifest(Options{})
	if err != nil {
		t.Fatalf("BuildManifest fallback: %v", err)
	}
	if m.Source.Selected != OriginUserHomeDotOpenClaw {
		t.Fatalf("selected = %q, want %q", m.Source.Selected, OriginUserHomeDotOpenClaw)
	}

	_, err = BuildManifest(Options{Source: filepath.Join(root, "missing")})
	if err == nil {
		t.Fatalf("missing explicit source should error")
	}
}

func findConfig(m *Manifest, key string) *ConfigEntry {
	for i := range m.Config {
		if m.Config[i].OpenClawKey == key {
			return &m.Config[i]
		}
	}
	return nil
}

func findEnv(m *Manifest, key string) *EnvEntry {
	for i := range m.Env {
		if m.Env[i].OpenClawKey == key {
			return &m.Env[i]
		}
	}
	return nil
}

func findFile(m *Manifest, kind string) *FileEntry {
	for i := range m.Files {
		if m.Files[i].Kind == kind {
			return &m.Files[i]
		}
	}
	return nil
}

func findSecretRef(m *Manifest, key string) *SecretRefEntry {
	for i := range m.SecretRefs {
		if m.SecretRefs[i].OpenClawKey == key {
			return &m.SecretRefs[i]
		}
	}
	return nil
}
