package providerconfig

import "testing"

func TestCustomProviderModelSwitchPatch_KeyEnvDoesNotSynthesizeAPIKey(t *testing.T) {
	ref := CustomProviderRef{Name: "acme", BaseURL: "https://acme.invalid/v1", KeyEnv: "ACME_KEY"}

	got := PlanCustomProviderModelSwitch(ref, "new-model")

	want := CustomProviderModelSwitchPatch{
		ProviderName: "acme",
		DefaultModel: "new-model",
		BaseURL:      "https://acme.invalid/v1",
		KeyEnv:       "ACME_KEY",
		Evidence:     "credential_write_skipped_key_env",
	}
	if got != want {
		t.Fatalf("PlanCustomProviderModelSwitch = %+v, want %+v", got, want)
	}
	if got.APIKey != "" {
		t.Fatalf("PlanCustomProviderModelSwitch synthesized api_key %q for key_env-backed provider", got.APIKey)
	}
}

func TestCustomProviderModelSwitchPatch_InlineEnvRefPreserved(t *testing.T) {
	ref := CustomProviderRef{Name: "acme", APIKey: "${ACME_KEY}"}

	got := PlanCustomProviderModelSwitch(ref, "new-model")

	want := CustomProviderModelSwitchPatch{
		ProviderName: "acme",
		DefaultModel: "new-model",
		APIKey:       "${ACME_KEY}",
		Evidence:     "credential_ref_preserved",
	}
	if got != want {
		t.Fatalf("PlanCustomProviderModelSwitch = %+v, want %+v", got, want)
	}
}

func TestCustomProviderModelSwitchPatch_PlaintextPreserved(t *testing.T) {
	ref := CustomProviderRef{Name: "acme", APIKey: "plain-existing-token"}

	got := PlanCustomProviderModelSwitch(ref, "new-model")

	want := CustomProviderModelSwitchPatch{
		ProviderName: "acme",
		DefaultModel: "new-model",
		APIKey:       "plain-existing-token",
		Evidence:     "plaintext_preserved",
	}
	if got != want {
		t.Fatalf("PlanCustomProviderModelSwitch = %+v, want %+v", got, want)
	}
}

func TestCustomProviderModelSwitchPatch_MissingCredentialStillUpdatesModelWithEvidence(t *testing.T) {
	ref := CustomProviderRef{Name: "acme"}

	got := PlanCustomProviderModelSwitch(ref, "new-model")

	want := CustomProviderModelSwitchPatch{
		ProviderName: "acme",
		DefaultModel: "new-model",
		Evidence:     "credential_missing",
	}
	if got != want {
		t.Fatalf("PlanCustomProviderModelSwitch = %+v, want %+v", got, want)
	}
}

func TestCustomProviderModelSwitchPatch_TrimsModelAndProviderFields(t *testing.T) {
	ref := CustomProviderRef{Name: " acme ", BaseURL: " https://acme.invalid/v1 ", APIKey: " ${ACME_KEY} "}

	got := PlanCustomProviderModelSwitch(ref, " new-model ")

	want := CustomProviderModelSwitchPatch{
		ProviderName: "acme",
		DefaultModel: "new-model",
		BaseURL:      "https://acme.invalid/v1",
		APIKey:       "${ACME_KEY}",
		Evidence:     "credential_ref_preserved",
	}
	if got != want {
		t.Fatalf("PlanCustomProviderModelSwitch = %+v, want %+v", got, want)
	}
}
