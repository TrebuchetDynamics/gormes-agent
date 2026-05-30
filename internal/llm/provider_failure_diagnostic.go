package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/providerdiagnostics"

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
	diagnostic := providerdiagnostics.Build(providerdiagnostics.Input{
		Provider:         input.Provider,
		Model:            input.Model,
		CredentialSource: input.CredentialSource,
		Classification:   providerDiagnosticClassification(classification),
	})
	return ProviderFailureDiagnostic{
		Provider:         diagnostic.Provider,
		Model:            diagnostic.Model,
		CredentialSource: diagnostic.CredentialSource,
		Kind:             ProviderErrorKind(diagnostic.Kind),
		Class:            errorClassFromString(diagnostic.Class),
		Status:           diagnostic.Status,
		Message:          diagnostic.Message,
		Retryable:        diagnostic.Retryable,
		NextAction:       diagnostic.NextAction,
		Redacted:         diagnostic.Redacted,
	}
}

func errorClassFromString(class string) ErrorClass {
	switch class {
	case ClassRetryable.String():
		return ClassRetryable
	case ClassFatal.String():
		return ClassFatal
	default:
		return ClassUnknown
	}
}
