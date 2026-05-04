package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSecretsAuditDetectsPlaintextUnresolvedRefsAndPrecedenceDrift(t *testing.T) {
	resolver := fakeSecretResolver{"GORMES_API_KEY": "sk-live-secret"}
	previous := &SecretsRuntimeSnapshot{
		Generation: 7,
		Entries: map[string]SecretsRuntimeEntry{
			"hermes.api_key": {
				Path: "hermes.api_key",
				Ref:  SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "OLD_API_KEY"},
			},
		},
	}

	result := AuditSecrets(context.Background(), SecretsAuditRequest{
		Resolver:         resolver,
		PreviousSnapshot: previous,
		Plan: SecretsPlan{Targets: []SecretTarget{
			{
				Path:      "hermes.api_key",
				Required:  true,
				Plaintext: "sk-live-secret",
				Ref:       SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "GORMES_API_KEY"},
			},
			{
				Path:     "telegram.bot_token",
				Required: true,
				Ref:      SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "MISSING_BOT_TOKEN"},
			},
		}},
	})

	if result.OK {
		t.Fatalf("audit OK = true, want findings")
	}
	for _, want := range []string{SecretsFindingPlaintext, SecretsFindingUnresolvedRef, SecretsFindingPrecedenceDrift} {
		if !secretsFindingCodePresent(result.Findings, want) {
			t.Fatalf("findings = %+v, missing %s", result.Findings, want)
		}
	}
	rendered := mustJSONSecrets(t, result)
	for _, leak := range []string{"sk-live-secret"} {
		if strings.Contains(rendered, leak) {
			t.Fatalf("audit result leaked secret %q:\n%s", leak, rendered)
		}
	}
}

func TestSecretsReloadAtomicallyKeepsLastGoodSnapshotOnFailure(t *testing.T) {
	controller := NewSecretsRuntimeController(SecretsRuntimeControllerConfig{
		Resolver: fakeSecretResolver{"GORMES_API_KEY": "sk-live-secret"},
		InitialSnapshot: &SecretsRuntimeSnapshot{
			Generation: 3,
			Entries: map[string]SecretsRuntimeEntry{
				"hermes.api_key": {
					Path:     "hermes.api_key",
					Ref:      SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "GORMES_API_KEY"},
					Resolved: true,
				},
			},
		},
	})

	goodPlan := SecretsPlan{Targets: []SecretTarget{{
		Path:     "hermes.api_key",
		Required: true,
		Ref:      SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "GORMES_API_KEY"},
	}}}
	reloaded, err := controller.Reload(context.Background(), goodPlan)
	if err != nil {
		t.Fatalf("Reload good plan: %v", err)
	}
	if reloaded.Code != SecretsEvidenceReloaded || !reloaded.Snapshot.Entries["hermes.api_key"].Resolved {
		t.Fatalf("good reload = %+v, want reloaded resolved snapshot", reloaded)
	}
	if got := controller.Snapshot().Generation; got != 4 {
		t.Fatalf("generation after good reload = %d, want 4", got)
	}

	badPlan := SecretsPlan{Targets: []SecretTarget{{
		Path:     "hermes.api_key",
		Required: true,
		Ref:      SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "MISSING_API_KEY"},
	}}}
	failed, err := controller.Reload(context.Background(), badPlan)
	if err == nil {
		t.Fatalf("Reload missing ref err = nil, result=%+v", failed)
	}
	if failed.Code != SecretsEvidenceUnavailable {
		t.Fatalf("failed reload code = %q, want %q", failed.Code, SecretsEvidenceUnavailable)
	}
	if failed.Snapshot.Generation != 4 || failed.Snapshot.Entries["hermes.api_key"].Ref.ID != "GORMES_API_KEY" {
		t.Fatalf("failed reload snapshot = %+v, want last good snapshot preserved", failed.Snapshot)
	}
	if got := controller.Snapshot().Generation; got != 4 {
		t.Fatalf("controller generation after failed reload = %d, want unchanged 4", got)
	}
	if strings.Contains(mustJSONSecrets(t, failed), "sk-live-secret") {
		t.Fatalf("failed reload output leaked secret:\n%s", mustJSONSecrets(t, failed))
	}
}

func TestSecretsConfigureProducesTypedRefAndPreflightsWithoutPersistingValue(t *testing.T) {
	result, err := ConfigureSecretRef(context.Background(), SecretsConfigureRequest{
		Resolver: fakeSecretResolver{"OPENROUTER_API_KEY": "sk-openrouter-secret"},
		Path:     "providers.openrouter.api_key",
		Required: true,
		Ref: SecretRef{
			Source:   SecretRefSourceEnv,
			Provider: DefaultSecretProviderAlias,
			ID:       "OPENROUTER_API_KEY",
		},
	})
	if err != nil {
		t.Fatalf("ConfigureSecretRef: %v", err)
	}
	if result.Code != SecretsEvidenceConfigured || !result.PreflightOK {
		t.Fatalf("configure result = %+v, want configured preflight OK", result)
	}
	if result.Target.Ref.Source != SecretRefSourceEnv || result.Target.Ref.Provider != DefaultSecretProviderAlias || result.Target.Ref.ID != "OPENROUTER_API_KEY" {
		t.Fatalf("configured ref = %+v, want typed source/provider/id", result.Target.Ref)
	}
	rendered := mustJSONSecrets(t, result)
	if strings.Contains(rendered, "sk-openrouter-secret") {
		t.Fatalf("configure output leaked secret:\n%s", rendered)
	}
}

func TestSecretsApplyResolvesGeneratedPlanWithoutPersistingPlaintext(t *testing.T) {
	controller := NewSecretsRuntimeController(SecretsRuntimeControllerConfig{
		Resolver: fakeSecretResolver{"GORMES_API_KEY": "sk-apply-secret"},
	})
	applied, err := controller.Apply(context.Background(), SecretsPlan{Targets: []SecretTarget{{
		Path:     "hermes.api_key",
		Required: true,
		Ref:      SecretRef{Source: SecretRefSourceEnv, Provider: DefaultSecretProviderAlias, ID: "GORMES_API_KEY"},
	}}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.Code != SecretsEvidenceApplied || applied.Snapshot.Generation != 1 {
		t.Fatalf("apply result = %+v, want generation 1 applied", applied)
	}
	if !applied.Snapshot.Entries["hermes.api_key"].Resolved {
		t.Fatalf("snapshot entry = %+v, want resolved", applied.Snapshot.Entries["hermes.api_key"])
	}
	if strings.Contains(mustJSONSecrets(t, applied), "sk-apply-secret") {
		t.Fatalf("apply output leaked secret:\n%s", mustJSONSecrets(t, applied))
	}
}

type fakeSecretResolver map[string]string

func (r fakeSecretResolver) ResolveSecretString(ref SecretRef) (string, SecretRefEvidence, error) {
	evidence := SecretRefEvidence{Source: ref.Source, Provider: ref.Provider, ID: ref.ID, Redacted: true}
	if value, ok := r[ref.ID]; ok && strings.TrimSpace(value) != "" {
		evidence.Code = "secret_ref_resolved"
		return value, evidence, nil
	}
	evidence.Code = "secret_ref_missing"
	return "", evidence, errFakeSecretMissing(ref.ID)
}

type errFakeSecretMissing string

func (e errFakeSecretMissing) Error() string { return "missing secret ref " + string(e) }

func secretsFindingCodePresent(findings []SecretsAuditFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func mustJSONSecrets(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	return string(raw)
}
