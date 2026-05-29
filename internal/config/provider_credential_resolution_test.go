package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderCredentialResolutionOrdersRouteEnvBeforeSecretRefAndManifest(t *testing.T) {
	lookup := mapProviderCredentialLookup(map[string]string{
		"ROUTE_KEY":          "route-secret",
		"SECRET_REF_KEY":     "ref-secret",
		"OPENROUTER_API_KEY": "manifest-secret",
	})

	resolution := ResolveProviderCredential(ProviderCredentialRequest{
		Provider:  "openrouter",
		APIKeyEnv: "ROUTE_KEY",
		APIKeyRef: &SecretRef{Source: SecretRefSourceEnv, ID: "SECRET_REF_KEY"},
		APIKey:    "inline-secret",
		LookupEnv: lookup,
		Secrets:   SecretsCfg{Defaults: SecretProviderDefaults{Env: DefaultSecretProviderAlias}},
	})

	if resolution.Status != ProviderCredentialConfigured || !resolution.Available || resolution.Value != "route-secret" {
		t.Fatalf("resolution = %+v, want route env credential first", resolution)
	}
	if resolution.Source != ProviderCredentialSourceEnv || resolution.Evidence.ID != "ROUTE_KEY" || !resolution.Evidence.Redacted {
		t.Fatalf("evidence = %+v source=%s, want redacted route env evidence", resolution.Evidence, resolution.Source)
	}
}

func TestProviderCredentialResolutionResolvesSecretRefMaterialAndEvidence(t *testing.T) {
	resolution := ResolveProviderCredential(ProviderCredentialRequest{
		Provider:  "openrouter",
		APIKeyRef: &SecretRef{Source: SecretRefSourceEnv, ID: "SECRET_REF_KEY"},
		LookupEnv: mapProviderCredentialLookup(map[string]string{"SECRET_REF_KEY": "ref-secret"}),
		Secrets:   SecretsCfg{Defaults: SecretProviderDefaults{Env: DefaultSecretProviderAlias}},
	})

	if resolution.Status != ProviderCredentialConfigured || !resolution.Available || resolution.Value != "ref-secret" {
		t.Fatalf("resolution = %+v, want resolved SecretRef material", resolution)
	}
	if resolution.Evidence.Code != SecretRefEvidenceResolved || resolution.Evidence.ID != "SECRET_REF_KEY" || !resolution.Evidence.Redacted {
		t.Fatalf("evidence = %+v, want redacted resolved SecretRef evidence", resolution.Evidence)
	}
}

func TestProviderCredentialResolutionReadsCredentialPoolWhenRequested(t *testing.T) {
	home := t.TempDir()
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: home, Provider: "openrouter"}, []PooledCredential{{
		ID:          "key-1",
		AuthType:    CredentialAuthAPIKey,
		AccessToken: "pool-secret",
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	resolution := ResolveProviderCredential(ProviderCredentialRequest{
		Provider:          "openrouter",
		CredentialHome:    home,
		UseCredentialPool: true,
		LookupEnv:         mapProviderCredentialLookup(nil),
	})

	if resolution.Status != ProviderCredentialConfigured || !resolution.Available || resolution.Value != "pool-secret" {
		t.Fatalf("resolution = %+v, want credential pool material", resolution)
	}
	if resolution.Source != ProviderCredentialSourcePool || resolution.PoolEvidence.Code != CredentialPoolEvidenceLoaded || !resolution.PoolEvidence.Redacted {
		t.Fatalf("pool evidence = %+v source=%s, want redacted pool evidence", resolution.PoolEvidence, resolution.Source)
	}
}

func TestProviderCredentialResolutionPreservesLocalOptionalNotRequired(t *testing.T) {
	resolution := ResolveProviderCredential(ProviderCredentialRequest{Provider: "custom", Local: true, Optional: true})
	if resolution.Status != ProviderCredentialNotRequired || !resolution.Available || resolution.Value != "" {
		t.Fatalf("resolution = %+v, want not-required without material", resolution)
	}
}

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

func mapProviderCredentialLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
