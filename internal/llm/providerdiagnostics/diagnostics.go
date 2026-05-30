package providerdiagnostics

import (
	"regexp"
	"strings"
)

// Classification is the provider-error context needed to produce an operator
// diagnostic without importing the root llm package.
type Classification struct {
	Kind      string
	Class     string
	Status    int
	Message   string
	Retryable bool
}

// Input is the safe context available when a provider call fails.
type Input struct {
	Provider         string
	Model            string
	CredentialSource string
	Classification   Classification
}

// Diagnostic is a redacted, action-oriented error envelope for provider,
// auth, and operator surfaces.
type Diagnostic struct {
	Provider         string
	Model            string
	CredentialSource string
	Kind             string
	Class            string
	Status           int
	Message          string
	Retryable        bool
	NextAction       string
	Redacted         bool
}

// Build creates a redacted diagnostic from an already-classified provider
// error. Classification remains owned by the root provider-error boundary.
func Build(input Input) Diagnostic {
	classification := input.Classification
	return Diagnostic{
		Provider:         diagnosticValue(input.Provider, "provider"),
		Model:            diagnosticValue(input.Model, "model"),
		CredentialSource: diagnosticValue(input.CredentialSource, "unknown"),
		Kind:             classification.Kind,
		Class:            classification.Class,
		Status:           classification.Status,
		Message:          diagnosticValue(classification.Message, "provider failure"),
		Retryable:        classification.Retryable,
		NextAction:       NextAction(classification),
		Redacted:         true,
	}
}

// NextAction maps a provider-error classification to the next operator action.
func NextAction(classification Classification) string {
	switch classification.Kind {
	case "auth":
		return "refresh_or_replace_provider_credential"
	case "rate_limit":
		return "retry_after_or_rotate_credential"
	case "context":
		return "compress_context_or_reduce_prompt"
	case "image_too_large":
		return "shrink_or_remove_image_input"
	case "retryable", "timeout":
		return "retry_with_bounded_backoff"
	case "non_retryable":
		return "inspect_provider_request_configuration"
	default:
		if classification.Class == "retryable" {
			return "retry_with_bounded_backoff"
		}
		if classification.Class == "fatal" {
			return "inspect_provider_auth_or_request"
		}
		return "inspect_provider_diagnostics"
	}
}

func diagnosticValue(value, fallback string) string {
	value = RedactText(value)
	if value == "" {
		return fallback
	}
	return value
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|token|secret)=([^&\s"']+)`),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]+`),
}

// RedactText removes common credential shapes and bounds diagnostic text.
func RedactText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, pattern := range secretPatterns {
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
