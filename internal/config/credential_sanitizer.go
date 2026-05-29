package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

const CredentialSanitizerEvidenceNonASCIIStripped = credentials.CredentialSanitizerEvidenceNonASCIIStripped

type CredentialSanitizerWarning = credentials.CredentialSanitizerWarning
type CredentialSanitizer = credentials.CredentialSanitizer

func NewCredentialSanitizer(recorder func(CredentialSanitizerWarning)) *CredentialSanitizer {
	return credentials.NewCredentialSanitizer(recorder)
}

func sanitizeCredentialValue(key, value string) string {
	return credentials.SanitizeCredentialValue(key, value)
}

func resetCredentialSanitizerWarnings() {
	credentials.ResetDefaultCredentialSanitizerWarnings()
}
