package gateway

import gatewayadmission "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/admission"

// StartupAdmissionInput captures the startup-level admission state before any
// channel connects. It is intentionally platform-neutral so command and
// channel packages can share the guard without importing each other.
type StartupAdmissionInput = gatewayadmission.StartupAdmissionInput

// AdmissionEvidence is redacted startup admission evidence.
type AdmissionEvidence = gatewayadmission.AdmissionEvidence

// CredentialGuardValue is one credential field belonging to a configured
// platform.
type CredentialGuardValue = gatewayadmission.CredentialGuardValue

// CredentialGuardPlatform describes one platform for startup placeholder-token
// rejection.
type CredentialGuardPlatform = gatewayadmission.CredentialGuardPlatform

// CredentialGuardReport contains disabled platform names plus redacted
// evidence. It never includes raw credential values.
type CredentialGuardReport = gatewayadmission.CredentialGuardReport

// CheckStartupAllowlist reports the Hermes-compatible startup warning when no
// allowlist and no explicit allow-all override is configured.
func CheckStartupAllowlist(input StartupAdmissionInput) []AdmissionEvidence {
	return gatewayadmission.CheckStartupAllowlist(input)
}

// CheckWeakCredentialPlatforms disables enabled platforms whose non-empty
// credentials are obvious placeholders.
func CheckWeakCredentialPlatforms(platforms []CredentialGuardPlatform) CredentialGuardReport {
	return gatewayadmission.CheckWeakCredentialPlatforms(platforms)
}

// WeakCredentialPlaceholder recognizes only explicit placeholder values. Empty
// credentials are handled by normal missing-credential paths.
func WeakCredentialPlaceholder(value string) bool {
	return gatewayadmission.WeakCredentialPlaceholder(value)
}
