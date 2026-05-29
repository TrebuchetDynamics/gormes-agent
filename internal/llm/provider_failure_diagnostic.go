package llm

import (
	"regexp"
	"strings"
)

// ProviderFailureDiagnosticInput is the safe context available when a provider
// call fails. CredentialSource should identify where the credential came from
// (env/config/pool), not the credential itself; this helper still redacts
// common secret shapes defensively before returning operator evidence.
type ProviderFailureDiagnosticInput struct {
	Provider         string
	Model            string
	CredentialSource string
	Err              error
}

// ProviderFailureDiagnostic is a redacted, action-oriented error envelope for
// provider/auth/operator surfaces.
type ProviderFailureDiagnostic struct {
	Provider         string
	Model            string
	CredentialSource string
	Kind             ProviderErrorKind
	Class            ErrorClass
	Status           int
	Message          string
	Retryable        bool
	NextAction       string
	Redacted         bool
}

// BuildProviderFailureDiagnostic classifies err and adds redacted provider,
// model, credential-source, and next-action evidence.
func BuildProviderFailureDiagnostic(input ProviderFailureDiagnosticInput) ProviderFailureDiagnostic {
	classification := ClassifyProviderError(input.Err)
	return ProviderFailureDiagnostic{
		Provider:         providerDiagnosticValue(input.Provider, "provider"),
		Model:            providerDiagnosticValue(input.Model, "model"),
		CredentialSource: providerDiagnosticValue(input.CredentialSource, "unknown"),
		Kind:             classification.Kind,
		Class:            classification.Class,
		Status:           classification.Status,
		Message:          providerDiagnosticValue(classification.Message, "provider failure"),
		Retryable:        classification.Retryable,
		NextAction:       providerFailureNextAction(classification),
		Redacted:         true,
	}
}

func providerFailureNextAction(classification ProviderErrorClassification) string {
	switch classification.Kind {
	case ProviderErrorAuth:
		return "refresh_or_replace_provider_credential"
	case ProviderErrorRateLimit:
		return "retry_after_or_rotate_credential"
	case ProviderErrorContext:
		return "compress_context_or_reduce_prompt"
	case ProviderErrorImageTooLarge:
		return "shrink_or_remove_image_input"
	case ProviderErrorRetryable, ProviderErrorTimeout:
		return "retry_with_bounded_backoff"
	case ProviderErrorNonRetryable:
		return "inspect_provider_request_configuration"
	default:
		if classification.Class == ClassRetryable {
			return "retry_with_bounded_backoff"
		}
		if classification.Class == ClassFatal {
			return "inspect_provider_auth_or_request"
		}
		return "inspect_provider_diagnostics"
	}
}

func providerDiagnosticValue(value, fallback string) string {
	value = redactProviderDiagnosticText(value)
	if value == "" {
		return fallback
	}
	return value
}

var providerDiagnosticSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|token|secret)=([^&\s"']+)`),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]+`),
}

func redactProviderDiagnosticText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, pattern := range providerDiagnosticSecretPatterns {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			if strings.Contains(strings.ToLower(match), "bearer ") {
				return "Bearer [redacted]"
			}
			if idx := strings.Index(match, "="); idx >= 0 {
				return match[:idx+1] + "[redacted]"
			}
			return "[redacted]"
		})
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 240 {
		return text[:227] + "...[truncated]"
	}
	return text
}
