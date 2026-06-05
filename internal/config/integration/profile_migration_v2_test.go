package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestProfileMigrationV2PlanDryRunRedactsAndKeepsAllProfiles(t *testing.T) {
	home := seedLegacyProfileMigrationHome(t)

	plan, err := PlanProfileConfigV2Migration(ProfileMigrationV2Options{Home: home})
	if err != nil {
		t.Fatalf("PlanProfileConfigV2Migration: %v", err)
	}
	if plan.NoOp {
		t.Fatal("plan.NoOp = true, want legacy migration plan")
	}
	if got, want := profileMigrationIDs(plan.ProfileAdditions), []string{"main", "tulin"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("profile additions = %#v, want %#v", got, want)
	}
	main := profileMigrationByID(t, plan.ProfileAdditions, "main")
	if !main.Enabled {
		t.Fatalf("main profile enabled = false, want true")
	}
	if got, want := main.Workspaces, []string{"/srv/main", "/srv/shared"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("main workspaces = %#v, want %#v", got, want)
	}
	if got, want := main.Providers, []string{"openrouter"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("main providers = %#v, want %#v", got, want)
	}
	if got, want := main.Channels, []string{"telegram"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("main channels = %#v, want %#v", got, want)
	}
	tulin := profileMigrationByID(t, plan.ProfileAdditions, "tulin")
	if got, want := tulin.Workspaces, []string{"/srv/tulin"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tulin workspaces = %#v, want %#v", got, want)
	}
	if got, want := tulin.Providers, []string{"openai"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tulin providers = %#v, want %#v", got, want)
	}
	if got, want := tulin.Channels, []string{"discord"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tulin channels = %#v, want %#v", got, want)
	}

	if got, want := profileMigrationCredentialIDs(plan.CredentialAdditions), []string{"main-openrouter", "main-telegram", "tulin-discord", "tulin-openai"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("credential additions = %#v, want %#v", got, want)
	}
	for _, credential := range plan.CredentialAdditions {
		if credential.SecretRef == nil {
			t.Fatalf("credential %s SecretRef = nil", credential.ID)
		}
		if credential.SecretRef.Source != SecretRefSourceEnv {
			t.Fatalf("credential %s SecretRef.Source = %q, want env", credential.ID, credential.SecretRef.Source)
		}
		if strings.Contains(credential.SecretRef.ID, "sk-") || strings.Contains(credential.SecretRef.ID, "bot-token") || strings.Contains(credential.SecretRef.ID, "discord-token") {
			t.Fatalf("credential %s SecretRef leaked secret-looking value: %+v", credential.ID, credential.SecretRef)
		}
	}
	if got, want := profileMigrationSecretTargets(plan.SecretMovements), []string{"GORMES_MAIN_OPENROUTER_API_KEY", "GORMES_MAIN_TELEGRAM_BOT_TOKEN", "GORMES_TULIN_DISCORD_TOKEN", "GORMES_TULIN_OPENAI_API_KEY"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("secret movement targets = %#v, want %#v", got, want)
	}
	for _, move := range plan.SecretMovements {
		if !move.Redacted {
			t.Fatalf("secret movement %+v not marked redacted", move)
		}
	}

	if plan.ActiveProfile != "tulin" {
		t.Fatalf("ActiveProfile = %q, want tulin", plan.ActiveProfile)
	}
	if !profileMigrationHasManualAction(plan.ManualActions, "legacy_active_profile_compatibility") {
		t.Fatalf("manual actions = %+v, want legacy_active_profile_compatibility", plan.ManualActions)
	}
	if got, want := profileMigrationReadCodes(plan.FallbackReads), []string{"active_profile", "env_file", "profile_config", "root_config"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback read codes = %#v, want %#v", got, want)
	}
	preview := strings.Join(plan.PreviewLines, "\n")
	for _, leaked := range []string{"sk-main-secret", "bot-token-main", "sk-tulin-secret", "discord-token-tulin"} {
		if strings.Contains(preview, leaked) {
			t.Fatalf("preview leaked %q:\n%s", leaked, preview)
		}
	}
	for _, want := range []string{"profiles.main", "profiles.tulin", "credentials.main-openrouter", "credentials.tulin-discord", "legacy active_profile=tulin"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q:\n%s", want, preview)
		}
	}
}

func TestProfileMigrationV2ApplyWritesRootBackupAndIsIdempotent(t *testing.T) {
	home := seedLegacyProfileMigrationHome(t)
	now := time.Date(2026, 5, 22, 12, 34, 56, 0, time.UTC)

	result, err := ApplyProfileConfigV2Migration(ProfileMigrationV2Options{Home: home, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("ApplyProfileConfigV2Migration: %v", err)
	}
	if !result.Wrote || result.NoOp {
		t.Fatalf("apply result = %+v, want wrote non-noop", result)
	}
	if result.BackupPath == "" {
		t.Fatalf("BackupPath empty in result %+v", result)
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Fatalf("backup stat %s: %v", result.BackupPath, err)
	}
	body, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read migrated config: %v", err)
	}
	text := string(body)
	for _, want := range []string{"config_version = 2", "[profiles.main]", "[profiles.tulin]", "[credentials.main-openrouter]", "[credentials.tulin-discord]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("migrated config missing %q:\n%s", want, text)
		}
	}
	for _, leaked := range []string{"sk-main-secret", "bot-token-main", "sk-tulin-secret", "discord-token-tulin", "active_profile"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("migrated config leaked or promoted %q:\n%s", leaked, text)
		}
	}
	legacyProfileConfig := filepath.Join(home, "profiles", "tulin", "config.toml")
	if _, err := os.Stat(legacyProfileConfig); err != nil {
		t.Fatalf("legacy profile config should be preserved, stat %s: %v", legacyProfileConfig, err)
	}

	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load migrated config: %v", err)
	}
	if !cfg.ProfileConfigV2Available() {
		t.Fatal("migrated config did not load as v2 profile config")
	}
	if got, want := enabledProfileIDs(cfg.EnabledProfileServices()), []string{"main", "tulin"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enabled profile IDs = %#v, want %#v", got, want)
	}

	second, err := ApplyProfileConfigV2Migration(ProfileMigrationV2Options{Home: home, Now: func() time.Time { return now.Add(time.Minute) }})
	if err != nil {
		t.Fatalf("second ApplyProfileConfigV2Migration: %v", err)
	}
	if !second.NoOp || second.Wrote || second.BackupPath != "" {
		t.Fatalf("second apply = %+v, want idempotent no-op without backup", second)
	}
}

func TestProfileMigrationV2ApplyRejectsCredentialConflictWithoutOverwrite(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "config.toml")
	body := []byte(`_config_version = 1

[hermes]
provider = "openrouter"
model = "openrouter/auto"
api_key = "sk-conflict-secret"

[credentials.main-openrouter]
kind = "channel"
channel = "telegram"
owner_profile = "main"
secret_ref = { source = "env", id = "EXISTING_TOKEN" }
`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	plan, err := PlanProfileConfigV2Migration(ProfileMigrationV2Options{Home: home})
	if err != nil {
		t.Fatalf("PlanProfileConfigV2Migration: %v", err)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0].ID != "main-openrouter" || plan.Conflicts[0].Resolution != "rename_or_skip" {
		t.Fatalf("conflicts = %+v, want main-openrouter rename_or_skip", plan.Conflicts)
	}
	_, err = ApplyProfileConfigV2Migration(ProfileMigrationV2Options{Home: home})
	if err == nil || !strings.Contains(err.Error(), "main-openrouter") {
		t.Fatalf("ApplyProfileConfigV2Migration conflict err = %v, want main-openrouter", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read post-conflict: %v", readErr)
	}
	if string(after) != string(body) {
		t.Fatalf("conflict apply rewrote config:\n%s", after)
	}
}

func seedLegacyProfileMigrationHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	root := []byte(`_config_version = 1

[hermes]
endpoint = "https://openrouter.ai/api/v1"
provider = "openrouter"
model = "openrouter/auto"
api_key = "sk-main-secret"

[telegram]
bot_token = "bot-token-main"
allowed_chat_id = 12345
allowed_user_ids = [6586915095]
tool_progress = "new"

[agents.defaults]
workspaces = ["/srv/main", "/srv/shared"]
channels = ["telegram"]
`)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), root, 0o600); err != nil {
		t.Fatalf("write root config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("GORMES_API_KEY=sk-env-existing\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "active_profile"), []byte("tulin\n"), 0o600); err != nil {
		t.Fatalf("write active_profile: %v", err)
	}
	profileDir := filepath.Join(home, "profiles", "tulin")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	profile := []byte(`_config_version = 1

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
`)
	if err := os.WriteFile(filepath.Join(profileDir, "config.toml"), profile, 0o600); err != nil {
		t.Fatalf("write profile config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "runtime.log"), []byte("preserve me"), 0o600); err != nil {
		t.Fatalf("write profile runtime data: %v", err)
	}
	return home
}

func profileMigrationIDs(profiles []ProfileMigrationV2ProfileAddition) []string {
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ID)
	}
	sort.Strings(ids)
	return ids
}

func profileMigrationByID(t *testing.T, profiles []ProfileMigrationV2ProfileAddition, id string) ProfileMigrationV2ProfileAddition {
	t.Helper()
	for _, profile := range profiles {
		if profile.ID == id {
			return profile
		}
	}
	t.Fatalf("profile %q missing from %+v", id, profiles)
	return ProfileMigrationV2ProfileAddition{}
}

func profileMigrationCredentialIDs(credentials []ProfileMigrationV2CredentialAddition) []string {
	ids := make([]string, 0, len(credentials))
	for _, credential := range credentials {
		ids = append(ids, credential.ID)
	}
	sort.Strings(ids)
	return ids
}

func profileMigrationSecretTargets(moves []ProfileMigrationV2SecretMovement) []string {
	ids := make([]string, 0, len(moves))
	for _, move := range moves {
		ids = append(ids, move.TargetEnv)
	}
	sort.Strings(ids)
	return ids
}

func profileMigrationHasManualAction(actions []ProfileMigrationV2ManualAction, code string) bool {
	for _, action := range actions {
		if action.Code == code {
			return true
		}
	}
	return false
}

func profileMigrationReadCodes(reads []ProfileMigrationV2FallbackRead) []string {
	codes := make([]string, 0, len(reads))
	seen := map[string]bool{}
	for _, read := range reads {
		if !seen[read.Code] {
			codes = append(codes, read.Code)
			seen[read.Code] = true
		}
	}
	sort.Strings(codes)
	return codes
}

func enabledProfileIDs(services []ProfileService) []string {
	ids := make([]string, 0, len(services))
	for _, service := range services {
		ids = append(ids, service.ID)
	}
	sort.Strings(ids)
	return ids
}
