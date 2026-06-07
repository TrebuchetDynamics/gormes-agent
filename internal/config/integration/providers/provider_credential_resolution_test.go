package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"testing"
)

func TestConfiguredProviderAuthPresentUsesSharedResolution(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{Provider: "openrouter"}, []PooledCredential{{
		ID:          "key-1",
		AuthType:    CredentialAuthAPIKey,
		AccessToken: "pool-secret",
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		t.Fatalf("auth store missing: %v", err)
	}

	cfg := Config{Hermes: HermesCfg{Provider: "openrouter"}}
	if !ConfiguredProviderAuthPresent(cfg) {
		t.Fatal("ConfiguredProviderAuthPresent = false, want true from credential pool")
	}
}
