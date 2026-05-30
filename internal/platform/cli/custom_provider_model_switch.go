package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/providerconfig"

// CustomProviderModelSwitchPatch is the pure write-shape for changing a named
// custom provider's default model without resolving or synthesizing secrets.
// It mirrors the fields a later config writer can persist while giving command
// and status surfaces auditable evidence about why credential bytes were, or
// were not, included in the patch.
type CustomProviderModelSwitchPatch = providerconfig.CustomProviderModelSwitchPatch

// PlanCustomProviderModelSwitch builds the config patch for switching a custom
// provider to targetModel. The helper is intentionally filesystem-, env-, and
// network-free: it preserves only credential storage already present in ref and
// never resolves key_env or ${VAR} references to plaintext.
func PlanCustomProviderModelSwitch(ref CustomProviderRef, targetModel string) CustomProviderModelSwitchPatch {
	return providerconfig.PlanCustomProviderModelSwitch(ref, targetModel)
}
