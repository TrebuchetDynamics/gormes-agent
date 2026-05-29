package cli

import "strings"

// CustomProviderModelSwitchPatch is the pure write-shape for changing a named
// custom provider's default model without resolving or synthesizing secrets.
// It mirrors the fields a later config writer can persist while giving command
// and status surfaces auditable evidence about why credential bytes were, or
// were not, included in the patch.
type CustomProviderModelSwitchPatch struct {
	ProviderName string
	DefaultModel string
	BaseURL      string
	APIKey       string
	KeyEnv       string
	Evidence     string
}

// PlanCustomProviderModelSwitch builds the config patch for switching a custom
// provider to targetModel. The helper is intentionally filesystem-, env-, and
// network-free: it preserves only credential storage already present in ref and
// never resolves key_env or ${VAR} references to plaintext.
func PlanCustomProviderModelSwitch(ref CustomProviderRef, targetModel string) CustomProviderModelSwitchPatch {
	apiKey := strings.TrimSpace(ref.APIKey)
	keyEnv := strings.TrimSpace(ref.KeyEnv)

	patch := CustomProviderModelSwitchPatch{
		ProviderName: strings.TrimSpace(ref.Name),
		DefaultModel: strings.TrimSpace(targetModel),
		BaseURL:      strings.TrimSpace(ref.BaseURL),
		KeyEnv:       keyEnv,
	}

	switch {
	case apiKey != "":
		patch.APIKey = apiKey
		if _, ok := envTemplateName(apiKey); ok {
			patch.Evidence = "credential_ref_preserved"
		} else {
			patch.Evidence = "plaintext_preserved"
		}
	case keyEnv != "":
		patch.Evidence = "credential_write_skipped_key_env"
	default:
		patch.Evidence = "credential_missing"
	}

	return patch
}
