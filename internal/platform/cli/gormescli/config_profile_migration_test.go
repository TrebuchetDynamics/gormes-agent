package gormescli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestConfigMigrateProfilesV2DryRunJSONPreviewsWithoutMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedConfigProfileMigrationLegacyHome(t)
	before := readConfigMigrationFile(t, config.ConfigPath())

	cmd := newConfigCommandForTest()
	stdout, stderr, err := executeConfigCommandForTest(cmd, "migrate", "--profiles-v2", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("config migrate --profiles-v2 --dry-run --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, "sk-main-secret") || strings.Contains(stdout, "bot-token-main") || strings.Contains(stdout, "sk-tulin-secret") || strings.Contains(stdout, "discord-token-tulin") {
		t.Fatalf("dry-run JSON leaked raw secret:\n%s", stdout)
	}
	var got struct {
		Mode             string   `json:"mode"`
		DryRun           bool     `json:"dry_run"`
		NoOp             bool     `json:"no_op"`
		Wrote            bool     `json:"wrote"`
		Profiles         []string `json:"profiles"`
		Credentials      []string `json:"credentials"`
		SecretEnvTargets []string `json:"secret_env_targets"`
		ManualActions    []string `json:"manual_actions"`
		Preview          []string `json:"preview"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("dry-run JSON must parse: %v\nstdout=%s", err, stdout)
	}
	if got.Mode != "profile_v2" || !got.DryRun || got.Wrote || got.NoOp {
		t.Fatalf("profile migration JSON flags = %+v, want mode=profile_v2 dry_run=true wrote=false no_op=false", got)
	}
	if want := []string{"main", "tulin"}; !reflect.DeepEqual(got.Profiles, want) {
		t.Fatalf("profiles = %#v, want %#v", got.Profiles, want)
	}
	if want := []string{"main-openrouter", "main-telegram", "tulin-discord", "tulin-openai"}; !reflect.DeepEqual(got.Credentials, want) {
		t.Fatalf("credentials = %#v, want %#v", got.Credentials, want)
	}
	if want := []string{"GORMES_MAIN_OPENROUTER_API_KEY", "GORMES_MAIN_TELEGRAM_BOT_TOKEN", "GORMES_TULIN_DISCORD_TOKEN", "GORMES_TULIN_OPENAI_API_KEY"}; !reflect.DeepEqual(got.SecretEnvTargets, want) {
		t.Fatalf("secret env targets = %#v, want %#v", got.SecretEnvTargets, want)
	}
	if !profileMigrationContainsString(got.ManualActions, "legacy_active_profile_compatibility") {
		t.Fatalf("manual actions = %#v, want legacy_active_profile_compatibility", got.ManualActions)
	}
	if !strings.Contains(strings.Join(got.Preview, "\n"), "profiles.tulin") {
		t.Fatalf("preview missing profiles.tulin evidence: %#v", got.Preview)
	}
	if after := readConfigMigrationFile(t, config.ConfigPath()); string(after) != string(before) {
		t.Fatalf("dry-run mutated config.toml:\nbefore=%s\nafter=%s", before, after)
	}
	if backups := globConfigMigrationBackups(t); len(backups) != 0 {
		t.Fatalf("dry-run created backups: %#v", backups)
	}
}

func TestConfigMigrateProfilesV2DryRunJSONReportsConflictsWithoutMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`_config_version = 1

[hermes]
provider = "openrouter"
model = "openrouter/auto"
api_key = "sk-conflict-secret"

[credentials.main-openrouter]
kind = "channel"
channel = "telegram"
owner_profile = "main"
secret_ref = { source = "env", id = "EXISTING_TOKEN" }
`))
	before := readConfigMigrationFile(t, config.ConfigPath())

	cmd := newConfigCommandForTest()
	stdout, stderr, err := executeConfigCommandForTest(cmd, "migrate", "--profiles-v2", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("config migrate --profiles-v2 conflict dry-run --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, "sk-conflict-secret") {
		t.Fatalf("conflict dry-run JSON leaked raw secret:\n%s", stdout)
	}
	var got struct {
		Mode      string   `json:"mode"`
		DryRun    bool     `json:"dry_run"`
		Wrote     bool     `json:"wrote"`
		Conflicts []string `json:"conflicts"`
		Preview   []string `json:"preview"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("conflict dry-run JSON must parse: %v\nstdout=%s", err, stdout)
	}
	if got.Mode != "profile_v2" || !got.DryRun || got.Wrote {
		t.Fatalf("conflict dry-run JSON flags = %+v, want mode=profile_v2 dry_run=true wrote=false", got)
	}
	if want := []string{"credential_id:main-openrouter"}; !reflect.DeepEqual(got.Conflicts, want) {
		t.Fatalf("conflicts = %#v, want %#v", got.Conflicts, want)
	}
	if !strings.Contains(strings.Join(got.Preview, "\n"), "conflict credential_id.main-openrouter requires rename_or_skip") {
		t.Fatalf("preview missing conflict resolution guidance: %#v", got.Preview)
	}
	if after := readConfigMigrationFile(t, config.ConfigPath()); string(after) != string(before) {
		t.Fatalf("conflict dry-run mutated config.toml:\nbefore=%s\nafter=%s", before, after)
	}
	if backups := globConfigMigrationBackups(t); len(backups) != 0 {
		t.Fatalf("conflict dry-run created backups: %#v", backups)
	}
}

func TestConfigMigrateProfilesV2ApplyJSONWritesBackupAndPreservesLegacyDirs(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedConfigProfileMigrationLegacyHome(t)

	cmd := newConfigCommandForTest()
	stdout, stderr, err := executeConfigCommandForTest(cmd, "migrate", "--profiles-v2", "--json")
	if err != nil {
		t.Fatalf("config migrate --profiles-v2 --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, "sk-main-secret") || strings.Contains(stdout, "bot-token-main") || strings.Contains(stdout, "sk-tulin-secret") || strings.Contains(stdout, "discord-token-tulin") {
		t.Fatalf("apply JSON leaked raw secret:\n%s", stdout)
	}
	var got struct {
		Mode       string   `json:"mode"`
		DryRun     bool     `json:"dry_run"`
		NoOp       bool     `json:"no_op"`
		Wrote      bool     `json:"wrote"`
		BackupPath string   `json:"backup_path"`
		Profiles   []string `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("apply JSON must parse: %v\nstdout=%s", err, stdout)
	}
	if got.Mode != "profile_v2" || got.DryRun || !got.Wrote || got.NoOp || got.BackupPath == "" {
		t.Fatalf("profile migration apply JSON = %+v, want wrote with backup", got)
	}
	if _, err := os.Stat(got.BackupPath); err != nil {
		t.Fatalf("backup stat %s: %v", got.BackupPath, err)
	}
	body := string(readConfigMigrationFile(t, config.ConfigPath()))
	for _, want := range []string{"config_version = 2", "[profiles.main]", "[profiles.tulin]", "[credentials.main-openrouter]", "[credentials.tulin-discord]"} {
		if !strings.Contains(body, want) {
			t.Fatalf("migrated config missing %q:\n%s", want, body)
		}
	}
	for _, leaked := range []string{"sk-main-secret", "bot-token-main", "sk-tulin-secret", "discord-token-tulin", "active_profile"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("migrated config leaked or promoted %q:\n%s", leaked, body)
		}
	}
	if _, err := os.Stat(filepath.Join(config.GormesHome(), "profiles", "tulin", "config.toml")); err != nil {
		t.Fatalf("legacy profile config should remain: %v", err)
	}
}

func seedConfigProfileMigrationLegacyHome(t *testing.T) {
	t.Helper()
	writeOneshotFlagConfig(t, []byte(`_config_version = 1

[hermes]
endpoint = "https://openrouter.ai/api/v1"
provider = "openrouter"
model = "openrouter/auto"
api_key = "sk-main-secret"

[telegram]
bot_token = "bot-token-main"
allowed_chat_id = 12345

[agents.defaults]
workspaces = ["/srv/main"]
channels = ["telegram"]
`))
	if err := os.WriteFile(filepath.Join(config.GormesHome(), "active_profile"), []byte("tulin\n"), 0o600); err != nil {
		t.Fatalf("write active_profile: %v", err)
	}
	profileDir := filepath.Join(config.GormesHome(), "profiles", "tulin")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "config.toml"), []byte(`_config_version = 1

[hermes]
provider = "openai"
model = "gpt-5.3-codex"
api_key = "sk-tulin-secret"

[discord]
token = "discord-token-tulin"
allowed_channel_id = "98765"

[agents.defaults]
workspace = "/srv/tulin"
channels = ["discord"]
`), 0o600); err != nil {
		t.Fatalf("write profile config: %v", err)
	}
}

func readConfigMigrationFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return body
}

func globConfigMigrationBackups(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(config.ConfigPath() + ".bak.*")
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	sort.Strings(matches)
	return matches
}

func profileMigrationContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
