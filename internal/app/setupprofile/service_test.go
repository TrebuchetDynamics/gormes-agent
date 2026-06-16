package setupprofile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProfileIDUsesMaterializedProfileRootName(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	if got := ProfileID(filepath.Join(base, "profiles", "work"), "main"); got != "work" {
		t.Fatalf("ProfileID named = %q, want work", got)
	}
	if got := ProfileID(filepath.Join(base, "home"), "main"); got != "main" {
		t.Fatalf("ProfileID default = %q, want main", got)
	}
}

func TestCredentialIDNormalizesChannel(t *testing.T) {
	if got := CredentialID(" Main ", "slack/app one"); got != "main-slack_app_one" {
		t.Fatalf("CredentialID = %q", got)
	}
}

func TestCompactStringsTrimsAndDeduplicates(t *testing.T) {
	got := CompactStrings([]string{"  a ", "", "b", "a"})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CompactStrings = %#v, want %#v", got, want)
	}
}

func TestInt64StringsFormatsValues(t *testing.T) {
	got := Int64Strings([]int64{1, 42})
	want := []string{"1", "42"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Int64Strings = %#v, want %#v", got, want)
	}
}

func TestChannelBindingMaterializedMainUsesMainProfileID(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	mainRoot := filepath.Join(base, "profiles", config.DefaultProfileID)
	if err := os.MkdirAll(mainRoot, 0o700); err != nil {
		t.Fatalf("mkdir materialized main profile: %v", err)
	}

	binding, err := WriteChannelBinding(mainRoot, base, config.DefaultProfileID, ChannelOptions{ChannelID: "telegram"})
	if err != nil {
		t.Fatalf("WriteChannelBinding: %v", err)
	}
	if binding.ProfileID != config.DefaultProfileID {
		t.Fatalf("binding profile id = %q, want main profile id %q", binding.ProfileID, config.DefaultProfileID)
	}
	if binding.CredentialID != "main-telegram" || binding.SecretEnvName != "GORMES_MAIN_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("binding = %+v, want main-profile credential naming", binding)
	}

	cfg, err := LoadRegistry(filepath.Join(base, "config.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if _, ok := cfg.Profiles[config.DefaultProfileID]; !ok {
		t.Fatalf("profiles.%s missing from gateway profile registry: %+v", config.DefaultProfileID, cfg.Profiles)
	}
	if _, ok := cfg.Profiles["default"]; ok {
		t.Fatalf("gateway setup created profiles.default for materialized main profile; want profiles.%s only: %+v", config.DefaultProfileID, cfg.Profiles)
	}
	if cred := cfg.Credentials["main-telegram"]; cred.Kind != "channel" || cred.Channel != "telegram" || cred.OwnerProfile != config.DefaultProfileID {
		t.Fatalf("credentials.main-telegram = %+v, want channel credential owned by %s", cred, config.DefaultProfileID)
	}
}

func TestChannelBindingNamedProfileUsesProfileRootName(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	workRoot := filepath.Join(base, "profiles", "work")
	if err := os.MkdirAll(workRoot, 0o700); err != nil {
		t.Fatalf("mkdir named profile: %v", err)
	}

	binding, err := WriteChannelBinding(workRoot, base, config.DefaultProfileID, ChannelOptions{ChannelID: "telegram"})
	if err != nil {
		t.Fatalf("WriteChannelBinding: %v", err)
	}
	if binding.ProfileID != "work" || binding.CredentialID != "work-telegram" || binding.SecretEnvName != "GORMES_WORK_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("binding = %+v, want work profile credential naming", binding)
	}
}
