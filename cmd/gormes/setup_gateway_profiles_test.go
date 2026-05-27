package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestSetupGatewayProfileBindingMaterializedDefaultUsesMainProfileID(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	defaultRoot := filepath.Join(base, "profiles", "default")
	if err := os.MkdirAll(defaultRoot, 0o700); err != nil {
		t.Fatalf("mkdir materialized default profile: %v", err)
	}
	t.Setenv("GORMES_HOME", defaultRoot)

	binding, err := writeSetupGatewayProfileChannelBinding(setupGatewayProfileChannelOptions{ChannelID: "telegram"})
	if err != nil {
		t.Fatalf("writeSetupGatewayProfileChannelBinding: %v", err)
	}
	if binding.ProfileID != config.DefaultProfileID {
		t.Fatalf("binding profile id = %q, want config default profile id %q", binding.ProfileID, config.DefaultProfileID)
	}
	if binding.CredentialID != "main-telegram" || binding.SecretEnvName != "GORMES_MAIN_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("binding = %+v, want main/default-profile credential naming", binding)
	}

	cfg, err := loadSetupGatewayProfileRegistry(filepath.Join(base, "config.toml"))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if _, ok := cfg.Profiles[config.DefaultProfileID]; !ok {
		t.Fatalf("profiles.%s missing from gateway profile registry: %+v", config.DefaultProfileID, cfg.Profiles)
	}
	if _, ok := cfg.Profiles["default"]; ok {
		t.Fatalf("gateway setup created profiles.default for materialized default profile; want profiles.%s only: %+v", config.DefaultProfileID, cfg.Profiles)
	}
	if cred := cfg.Credentials["main-telegram"]; cred.Kind != "channel" || cred.Channel != "telegram" || cred.OwnerProfile != config.DefaultProfileID {
		t.Fatalf("credentials.main-telegram = %+v, want channel credential owned by %s", cred, config.DefaultProfileID)
	}
}

func TestSetupGatewayProfileBindingNamedProfileUsesProfileRootName(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	workRoot := filepath.Join(base, "profiles", "work")
	if err := os.MkdirAll(workRoot, 0o700); err != nil {
		t.Fatalf("mkdir named profile: %v", err)
	}
	t.Setenv("GORMES_HOME", workRoot)

	binding, err := writeSetupGatewayProfileChannelBinding(setupGatewayProfileChannelOptions{ChannelID: "telegram"})
	if err != nil {
		t.Fatalf("writeSetupGatewayProfileChannelBinding: %v", err)
	}
	if binding.ProfileID != "work" || binding.CredentialID != "work-telegram" || binding.SecretEnvName != "GORMES_WORK_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("binding = %+v, want work profile credential naming", binding)
	}
}
