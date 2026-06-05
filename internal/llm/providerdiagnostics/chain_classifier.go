package providerdiagnostics

// ChainDecision tells the dispatcher what to do after a provider attempt.
type ChainDecision string

const (
	ChainDecisionRetry    ChainDecision = "retry"
	ChainDecisionFallback ChainDecision = "fallback"
	ChainDecisionAbort    ChainDecision = "abort"
	ChainDecisionSuccess  ChainDecision = "success"
)

// DefaultChainErrorClassifier maps provider classifications to retry,
// fallback, or abort decisions.
type DefaultChainErrorClassifier struct {
	MaxRetriesPerProvider int
}

// NewDefaultChainErrorClassifier creates a classifier with sensible defaults.
func NewDefaultChainErrorClassifier() *DefaultChainErrorClassifier {
	return &DefaultChainErrorClassifier{MaxRetriesPerProvider: 1}
}

// Decide maps a provider error classification to a chain decision.
//
// Rules:
//   - Fatal/auth errors → abort (no point trying other providers with same creds)
//   - Context too large → abort (other providers will also reject)
//   - Rate limit / timeout / retryable → retry up to MaxRetriesPerProvider, then fallback
//   - Image too large → fallback (different provider may accept)
//   - Non-retryable 4xx → fallback
//   - Unknown → fallback
func (c *DefaultChainErrorClassifier) Decide(classification Classification, attemptNumber int) ChainDecision {
	switch classification.Kind {
	case "auth":
		return ChainDecisionAbort
	case "context":
		return ChainDecisionAbort
	case "rate_limit":
		if attemptNumber < c.MaxRetriesPerProvider {
			return ChainDecisionRetry
		}
		return ChainDecisionFallback
	case "timeout", "retryable":
		if attemptNumber < c.MaxRetriesPerProvider {
			return ChainDecisionRetry
		}
		return ChainDecisionFallback
	case "image_too_large":
		return ChainDecisionFallback
	case "non_retryable":
		return ChainDecisionFallback
	default:
		return ChainDecisionFallback
	}
}
