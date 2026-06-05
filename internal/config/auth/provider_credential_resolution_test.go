package auth

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
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
		APIKeyRef: &credentials.SecretRef{Source: credentials.SecretRefSourceEnv, ID: "SECRET_REF_KEY"},
		APIKey:    "inline-secret",
		LookupEnv: lookup,
		Secrets:   credentials.SecretsCfg{Defaults: credentials.SecretProviderDefaults{Env: credentials.DefaultSecretProviderAlias}},
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
		APIKeyRef: &credentials.SecretRef{Source: credentials.SecretRefSourceEnv, ID: "SECRET_REF_KEY"},
		LookupEnv: mapProviderCredentialLookup(map[string]string{"SECRET_REF_KEY": "ref-secret"}),
		Secrets:   credentials.SecretsCfg{Defaults: credentials.SecretProviderDefaults{Env: credentials.DefaultSecretProviderAlias}},
	})

	if resolution.Status != ProviderCredentialConfigured || !resolution.Available || resolution.Value != "ref-secret" {
		t.Fatalf("resolution = %+v, want resolved SecretRef material", resolution)
	}
	if resolution.Evidence.Code != credentials.SecretRefEvidenceResolved || resolution.Evidence.ID != "SECRET_REF_KEY" || !resolution.Evidence.Redacted {
		t.Fatalf("evidence = %+v, want redacted resolved SecretRef evidence", resolution.Evidence)
	}
}

func TestProviderCredentialResolutionReadsCredentialPoolWhenRequested(t *testing.T) {
	home := t.TempDir()
	if err := credentials.SaveCredentialPoolEntries(credentials.CredentialPoolOptions{HermesHome: home, Provider: "openrouter"}, []credentials.PooledCredential{{
		ID:          "key-1",
		AuthType:    credentials.CredentialAuthAPIKey,
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
	if resolution.Source != ProviderCredentialSourcePool || resolution.PoolEvidence.Code != credentials.CredentialPoolEvidenceLoaded || !resolution.PoolEvidence.Redacted {
		t.Fatalf("pool evidence = %+v source=%s, want redacted pool evidence", resolution.PoolEvidence, resolution.Source)
	}
}

func TestProviderCredentialResolutionPassesProfileFilterToPool(t *testing.T) {
	home := t.TempDir()
	if err := credentials.SaveCredentialPoolEntries(credentials.CredentialPoolOptions{HermesHome: home, Provider: "openrouter"}, []credentials.PooledCredential{
		{ID: "beta", AuthType: credentials.CredentialAuthAPIKey, OwnerProfile: "beta", AccessToken: "beta-secret"},
		{ID: "alpha", AuthType: credentials.CredentialAuthAPIKey, Priority: 1, OwnerProfile: "alpha", AccessToken: "alpha-secret"},
	}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	resolution := ResolveProviderCredential(ProviderCredentialRequest{
		Provider:          "openrouter",
		ProfileID:         "alpha",
		CredentialHome:    home,
		UseCredentialPool: true,
		LookupEnv:         mapProviderCredentialLookup(nil),
	})

	if resolution.Status != ProviderCredentialConfigured || !resolution.Available || resolution.Value != "alpha-secret" {
		t.Fatalf("resolution = %+v, want alpha-owned credential", resolution)
	}
	if resolution.PoolEvidence.Count != 2 || resolution.PoolEvidence.FilteredCount != 1 {
		t.Fatalf("pool evidence = %+v, want profile-filtered evidence", resolution.PoolEvidence)
	}
}

func TestProviderCredentialResolutionPreservesLocalOptionalNotRequired(t *testing.T) {
	resolution := ResolveProviderCredential(ProviderCredentialRequest{Provider: "custom", Local: true, Optional: true})
	if resolution.Status != ProviderCredentialNotRequired || !resolution.Available || resolution.Value != "" {
		t.Fatalf("resolution = %+v, want not-required without material", resolution)
	}
}

func mapProviderCredentialLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
