package allowlist

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/admission/evidence"

// StartupAdmissionInput captures the startup-level admission state before any
// channel connects. It is intentionally platform-neutral so command and
// channel packages can share the guard without importing each other.
type StartupAdmissionInput struct {
	AllowlistConfigured bool
	AllowAll            bool
}

// AdmissionEvidence is redacted startup admission evidence.
type AdmissionEvidence = evidence.AdmissionEvidence

// CheckStartupAllowlist reports the Hermes-compatible startup warning when no
// allowlist and no explicit allow-all override is configured.
func CheckStartupAllowlist(input StartupAdmissionInput) []AdmissionEvidence {
	if input.AllowlistConfigured || input.AllowAll {
		return nil
	}
	return []AdmissionEvidence{{
		Code:    "gateway_allowlist_missing",
		Message: "gateway has no configured allowlist or explicit allow-all override",
	}}
}
