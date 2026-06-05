package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/providerdiagnostics"

// ChainDecision tells the dispatcher what to do after a provider attempt.
type ChainDecision = providerdiagnostics.ChainDecision

const (
	ChainDecisionRetry    = providerdiagnostics.ChainDecisionRetry
	ChainDecisionFallback = providerdiagnostics.ChainDecisionFallback
	ChainDecisionAbort    = providerdiagnostics.ChainDecisionAbort
	ChainDecisionSuccess  = providerdiagnostics.ChainDecisionSuccess
)

// ChainErrorClassifier decides whether a failed provider attempt should retry,
// fall back to the next provider, or abort the entire chain.
type ChainErrorClassifier interface {
	Decide(classification ProviderErrorClassification, attemptNumber int) ChainDecision
}

// DefaultChainErrorClassifier uses the existing ProviderErrorClassification
// to make chain-aware decisions.
type DefaultChainErrorClassifier struct {
	MaxRetriesPerProvider int
}

// NewDefaultChainErrorClassifier creates a classifier with sensible defaults.
func NewDefaultChainErrorClassifier() *DefaultChainErrorClassifier {
	return &DefaultChainErrorClassifier{
		MaxRetriesPerProvider: 1,
	}
}

// Decide maps a provider error classification to a chain decision.
func (c *DefaultChainErrorClassifier) Decide(classification ProviderErrorClassification, attemptNumber int) ChainDecision {
	inner := providerdiagnostics.DefaultChainErrorClassifier{MaxRetriesPerProvider: c.MaxRetriesPerProvider}
	return inner.Decide(providerDiagnosticClassification(classification), attemptNumber)
}

func providerDiagnosticClassification(classification ProviderErrorClassification) providerdiagnostics.Classification {
	return providerdiagnostics.Classification{
		Kind:      classification.Kind.String(),
		Class:     classification.Class.String(),
		Status:    classification.Status,
		Message:   classification.Message,
		Retryable: classification.Retryable,
	}
}
