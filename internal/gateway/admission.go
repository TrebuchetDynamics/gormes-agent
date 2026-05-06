package gateway

import "strings"

// StartupAdmissionInput captures the startup-level admission state before any
// channel connects. It is intentionally platform-neutral so command and
// channel packages can share the guard without importing each other.
type StartupAdmissionInput struct {
	AllowlistConfigured bool
	AllowAll            bool
}

// AdmissionEvidence is redacted startup admission evidence.
type AdmissionEvidence struct {
	Code     string
	Platform string
	Field    string
	Message  string
	Secret   string
}

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

// CredentialGuardValue is one credential field belonging to a configured
// platform.
type CredentialGuardValue struct {
	Field string
	Value string
}

// CredentialGuardPlatform describes one platform for startup placeholder-token
// rejection.
type CredentialGuardPlatform struct {
	Name        string
	Enabled     bool
	Credentials []CredentialGuardValue
}

// CredentialGuardReport contains disabled platform names plus redacted
// evidence. It never includes raw credential values.
type CredentialGuardReport struct {
	DisabledPlatforms []string
	Evidence          []AdmissionEvidence
}

// CheckWeakCredentialPlatforms disables enabled platforms whose non-empty
// credentials are obvious placeholders.
func CheckWeakCredentialPlatforms(platforms []CredentialGuardPlatform) CredentialGuardReport {
	var report CredentialGuardReport
	for _, platform := range platforms {
		name := strings.TrimSpace(platform.Name)
		if name == "" || !platform.Enabled {
			continue
		}
		for _, credential := range platform.Credentials {
			if !WeakCredentialPlaceholder(credential.Value) {
				continue
			}
			field := strings.TrimSpace(credential.Field)
			report.DisabledPlatforms = append(report.DisabledPlatforms, name)
			report.Evidence = append(report.Evidence, AdmissionEvidence{
				Code:     "gateway_weak_credential_disabled",
				Platform: name,
				Field:    field,
				Message:  "enabled platform disabled because a placeholder credential was configured",
			})
			break
		}
	}
	return report
}

// WeakCredentialPlaceholder recognizes only explicit placeholder values. Empty
// credentials are handled by normal missing-credential paths.
func WeakCredentialPlaceholder(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "***", "changeme", "your_api_key", "placeholder":
		return true
	default:
		return false
	}
}
