package credentials

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials/sanitization"

const CredentialSanitizerEvidenceNonASCIIStripped = sanitization.CredentialSanitizerEvidenceNonASCIIStripped

type CredentialSanitizerWarning = sanitization.CredentialSanitizerWarning
type CredentialSanitizer = sanitization.CredentialSanitizer

func NewCredentialSanitizer(recorder func(CredentialSanitizerWarning)) *CredentialSanitizer {
	return sanitization.NewCredentialSanitizer(recorder)
}

func SanitizeCredentialValue(key, value string) string {
	return sanitization.SanitizeCredentialValue(key, value)
}

func sanitizeCredentialValue(key, value string) string {
	return SanitizeCredentialValue(key, value)
}

func ResetDefaultCredentialSanitizerWarnings() {
	sanitization.ResetDefaultCredentialSanitizerWarnings()
}
