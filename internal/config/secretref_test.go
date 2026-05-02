package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretRefResolveEnvDefaultProvider(t *testing.T) {
	resolver := NewSecretResolver(SecretResolverConfig{
		Env: map[string]string{"OPENAI_API_KEY": "sk-live-secret"},
	})

	got, evidence, err := resolver.ResolveString(SecretRef{
		Source:   SecretRefSourceEnv,
		Provider: DefaultSecretProviderAlias,
		ID:       "OPENAI_API_KEY",
	})
	if err != nil {
		t.Fatalf("ResolveString env: %v", err)
	}
	if got != "sk-live-secret" {
		t.Fatalf("resolved env secret = %q, want secret value", got)
	}
	if evidence.Code != SecretRefEvidenceResolved || !evidence.Redacted {
		t.Fatalf("evidence = %+v, want resolved redacted", evidence)
	}

	_, evidence, err = resolver.ResolveString(SecretRef{
		Source:   SecretRefSourceEnv,
		Provider: DefaultSecretProviderAlias,
		ID:       "MISSING_API_KEY",
	})
	if err == nil {
		t.Fatal("ResolveString missing env err = nil, want fail-closed error")
	}
	if evidence.Code != SecretRefEvidenceMissing {
		t.Fatalf("missing evidence = %+v, want %s", evidence, SecretRefEvidenceMissing)
	}
	if strings.Contains(err.Error(), "sk-live-secret") {
		t.Fatalf("missing env error leaked unrelated secret: %v", err)
	}
}

func TestSecretRefResolveFileProviderJSONPointer(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secrets.json")
	if err := os.WriteFile(secretFile, []byte(`{"providers":{"openai":{"apiKey":"sk-file-secret"}}}`), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	resolver := NewSecretResolver(SecretResolverConfig{
		Secrets: SecretsCfg{Providers: map[string]SecretProviderCfg{
			"filemain": {Source: SecretRefSourceFile, Path: secretFile, Mode: SecretProviderModeJSON},
		}},
	})

	got, evidence, err := resolver.ResolveString(SecretRef{
		Source:   SecretRefSourceFile,
		Provider: "filemain",
		ID:       "/providers/openai/apiKey",
	})
	if err != nil {
		t.Fatalf("ResolveString file: %v", err)
	}
	if got != "sk-file-secret" {
		t.Fatalf("resolved file secret = %q, want file value", got)
	}
	if evidence.Code != SecretRefEvidenceResolved || !evidence.Redacted {
		t.Fatalf("evidence = %+v, want resolved redacted", evidence)
	}
}

func TestSecretRefFileProviderRejectsInsecureOrRelativePaths(t *testing.T) {
	dir := t.TempDir()
	insecure := filepath.Join(dir, "secrets.json")
	if err := os.WriteFile(insecure, []byte(`{"token":"sk-readable"}`), 0o644); err != nil {
		t.Fatalf("write insecure secret file: %v", err)
	}
	resolver := NewSecretResolver(SecretResolverConfig{
		Secrets: SecretsCfg{Providers: map[string]SecretProviderCfg{
			"filemain": {Source: SecretRefSourceFile, Path: insecure, Mode: SecretProviderModeJSON},
			"relative": {Source: SecretRefSourceFile, Path: "secrets.json", Mode: SecretProviderModeJSON, AllowInsecurePath: true},
		}},
	})

	_, evidence, err := resolver.ResolveString(SecretRef{Source: SecretRefSourceFile, Provider: "filemain", ID: "/token"})
	if err == nil {
		t.Fatal("ResolveString insecure file err = nil, want fail-closed error")
	}
	if evidence.Code != SecretRefEvidenceInsecurePath {
		t.Fatalf("insecure evidence = %+v, want %s", evidence, SecretRefEvidenceInsecurePath)
	}
	if strings.Contains(err.Error(), "sk-readable") {
		t.Fatalf("insecure path error leaked file secret: %v", err)
	}

	_, evidence, err = resolver.ResolveString(SecretRef{Source: SecretRefSourceFile, Provider: "relative", ID: "/token"})
	if err == nil {
		t.Fatal("ResolveString relative file err = nil, want fail-closed error")
	}
	if evidence.Code != SecretRefEvidenceInsecurePath {
		t.Fatalf("relative evidence = %+v, want %s", evidence, SecretRefEvidenceInsecurePath)
	}
}

func TestSecretRefConfigLoadsProvidersFromTOML(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[secrets.defaults]
env = "default"
file = "filemain"

[secrets.providers.filemain]
source = "file"
path = "/tmp/gormes-secrets.json"
mode = "json"
allow_insecure_path = true
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Secrets.Defaults.File != "filemain" {
		t.Fatalf("Secrets.Defaults.File = %q, want filemain", cfg.Secrets.Defaults.File)
	}
	provider := cfg.Secrets.Providers["filemain"]
	if provider.Source != SecretRefSourceFile || provider.Path != "/tmp/gormes-secrets.json" || provider.Mode != SecretProviderModeJSON || !provider.AllowInsecurePath {
		t.Fatalf("Secrets provider filemain = %+v", provider)
	}
}
