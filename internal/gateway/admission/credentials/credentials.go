package credentials

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/admission/evidence"
)

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
	Evidence          []evidence.AdmissionEvidence
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
			report.Evidence = append(report.Evidence, evidence.AdmissionEvidence{
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
