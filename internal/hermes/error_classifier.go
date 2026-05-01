package hermes

// ChainDecision tells the dispatcher what to do after a provider attempt.
type ChainDecision string

const (
	ChainDecisionRetry    ChainDecision = "retry"
	ChainDecisionFallback ChainDecision = "fallback"
	ChainDecisionAbort    ChainDecision = "abort"
	ChainDecisionSuccess  ChainDecision = "success"
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
//
// Rules:
//   - Fatal/auth errors → abort (no point trying other providers with same creds)
//   - Context too large → abort (other providers will also reject)
//   - Rate limit / retryable → retry up to MaxRetriesPerProvider, then fallback
//   - Image too large → fallback (different provider may accept)
//   - Non-retryable 4xx → fallback
//   - Unknown → fallback
func (c *DefaultChainErrorClassifier) Decide(classification ProviderErrorClassification, attemptNumber int) ChainDecision {
	switch classification.Kind {
	case ProviderErrorAuth:
		return ChainDecisionAbort
	case ProviderErrorContext:
		return ChainDecisionAbort
	case ProviderErrorRateLimit:
		if attemptNumber < c.MaxRetriesPerProvider {
			return ChainDecisionRetry
		}
		return ChainDecisionFallback
	case ProviderErrorRetryable:
		if attemptNumber < c.MaxRetriesPerProvider {
			return ChainDecisionRetry
		}
		return ChainDecisionFallback
	case ProviderErrorImageTooLarge:
		return ChainDecisionFallback
	case ProviderErrorNonRetryable:
		return ChainDecisionFallback
	default:
		return ChainDecisionFallback
	}
}
