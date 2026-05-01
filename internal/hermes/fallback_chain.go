package hermes

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ChainAttemptEvidence records what happened during a single provider attempt.
type ChainAttemptEvidence struct {
	Provider   string        `json:"provider"`
	StartTime  time.Time     `json:"start_time"`
	Duration   time.Duration `json:"duration"`
	ErrorKind  string        `json:"error_kind,omitempty"`
	Decision   ChainDecision `json:"decision"`
	ErrorMsg   string        `json:"error_msg,omitempty"`
	AttemptNum int           `json:"attempt_num"`
}

// ChainResult surfaces the outcome of a full chain dispatch.
type ChainResult struct {
	Success         bool                   `json:"success"`
	UsedProvider    string                 `json:"used_provider,omitempty"`
	Attempts        []ChainAttemptEvidence `json:"attempts"`
	FailedProviders []string               `json:"failed_providers"`
	FinalError      error                  `json:"-"`
	FinalErrorMsg   string                 `json:"final_error_msg,omitempty"`
}

// ProviderFactory creates a Client for a named provider.
type ProviderFactory interface {
	NewClient(providerName string) (Client, error)
}

// SimpleProviderFactory maps provider names to HTTP client construction.
type SimpleProviderFactory struct {
	BaseURLFor func(providerName string) string
	APIKeyFor  func(providerName string) string
}

// NewClient creates a Client for the given provider name.
func (f *SimpleProviderFactory) NewClient(providerName string) (Client, error) {
	baseURL := ""
	apiKey := ""
	if f.BaseURLFor != nil {
		baseURL = f.BaseURLFor(providerName)
	}
	if f.APIKeyFor != nil {
		apiKey = f.APIKeyFor(providerName)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("no base URL configured for provider %q", providerName)
	}
	return NewHTTPClientWithProvider(baseURL, apiKey, providerName), nil
}

// ProviderChain holds the ordered list of providers and dispatch configuration.
type ProviderChain struct {
	Order             []string
	Classifier        ChainErrorClassifier
	Factory           ProviderFactory
	PerAttemptTimeout time.Duration
}

// NewProviderChain creates a chain with the standard provider order.
func NewProviderChain(factory ProviderFactory) *ProviderChain {
	return &ProviderChain{
		Order: []string{
			"deepseek",
			"openai",
			"anthropic",
			"grok",
			"ollama",
		},
		Classifier:        NewDefaultChainErrorClassifier(),
		Factory:           factory,
		PerAttemptTimeout: 60 * time.Second,
	}
}

// Dispatch sends the ChatRequest through the provider chain.
func (c *ProviderChain) Dispatch(ctx context.Context, req ChatRequest) (Stream, *ChainResult, error) {
	result := &ChainResult{
		Attempts:        make([]ChainAttemptEvidence, 0),
		FailedProviders: make([]string, 0),
	}

	for _, providerName := range c.Order {
		stream, aborted := c.tryProvider(ctx, providerName, req, result)
		if stream != nil {
			result.Success = true
			result.UsedProvider = providerName
			return stream, result, nil
		}
		if aborted {
			break
		}
	}

	result.Success = false
	if len(result.Attempts) > 0 {
		last := result.Attempts[len(result.Attempts)-1]
		result.FinalErrorMsg = last.ErrorMsg
		result.FinalError = fmt.Errorf("provider chain exhausted; last error from %s: %s", last.Provider, last.ErrorMsg)
	} else {
		result.FinalError = fmt.Errorf("provider chain exhausted with no attempts")
	}
	return nil, result, result.FinalError
}

func (c *ProviderChain) tryProvider(ctx context.Context, providerName string, req ChatRequest, result *ChainResult) (Stream, bool) {
	client, err := c.Factory.NewClient(providerName)
	if err != nil {
		result.Attempts = append(result.Attempts, ChainAttemptEvidence{
			Provider:   providerName,
			StartTime:  time.Now(),
			Decision:   ChainDecisionFallback,
			ErrorMsg:   err.Error(),
			ErrorKind:  string(ProviderErrorUnknown),
			AttemptNum: 1,
		})
		result.FailedProviders = append(result.FailedProviders, providerName)
		return nil, false
	}

	for attemptNum := 1; ; attemptNum++ {
		start := time.Now()

		attemptCtx, cancel := ctx, func() {}
		if c.PerAttemptTimeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, c.PerAttemptTimeout)
		}

		stream, err := client.OpenStream(attemptCtx, req)
		cancel()
		duration := time.Since(start)

		if err == nil {
			result.Attempts = append(result.Attempts, ChainAttemptEvidence{
				Provider:   providerName,
				StartTime:  start,
				Duration:   duration,
				Decision:   ChainDecisionSuccess,
				AttemptNum: attemptNum,
			})
			return stream, false
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			result.Attempts = append(result.Attempts, ChainAttemptEvidence{
				Provider:   providerName,
				StartTime:  start,
				Duration:   duration,
				ErrorKind:  string(ProviderErrorUnknown),
				Decision:   ChainDecisionAbort,
				ErrorMsg:   err.Error(),
				AttemptNum: attemptNum,
			})
			result.FailedProviders = append(result.FailedProviders, providerName)
			return nil, true
		}

		classification := ClassifyProviderError(err)
		decision := c.Classifier.Decide(classification, attemptNum)

		result.Attempts = append(result.Attempts, ChainAttemptEvidence{
			Provider:   providerName,
			StartTime:  start,
			Duration:   duration,
			ErrorKind:  classification.Kind.String(),
			Decision:   decision,
			ErrorMsg:   err.Error(),
			AttemptNum: attemptNum,
		})

		if decision == ChainDecisionAbort {
			result.FailedProviders = append(result.FailedProviders, providerName)
			return nil, true
		}
		if decision == ChainDecisionFallback {
			result.FailedProviders = append(result.FailedProviders, providerName)
			return nil, false
		}
	}
}
