package providerdiagnostics

import "testing"

func TestDefaultChainErrorClassifierTimeoutRetriesThenFallback(t *testing.T) {
	classifier := &DefaultChainErrorClassifier{MaxRetriesPerProvider: 2}
	classification := Classification{Kind: "timeout"}

	if got := classifier.Decide(classification, 0); got != ChainDecisionRetry {
		t.Fatalf("attempt 0 decision = %q, want %q", got, ChainDecisionRetry)
	}
	if got := classifier.Decide(classification, 1); got != ChainDecisionRetry {
		t.Fatalf("attempt 1 decision = %q, want %q", got, ChainDecisionRetry)
	}
	if got := classifier.Decide(classification, 2); got != ChainDecisionFallback {
		t.Fatalf("attempt 2 decision = %q, want %q", got, ChainDecisionFallback)
	}
}

func TestDefaultChainErrorClassifierAbortAndFallbackKinds(t *testing.T) {
	classifier := NewDefaultChainErrorClassifier()
	if got := classifier.Decide(Classification{Kind: "auth"}, 0); got != ChainDecisionAbort {
		t.Fatalf("auth decision = %q, want %q", got, ChainDecisionAbort)
	}
	if got := classifier.Decide(Classification{Kind: "image_too_large"}, 0); got != ChainDecisionFallback {
		t.Fatalf("image decision = %q, want %q", got, ChainDecisionFallback)
	}
}
