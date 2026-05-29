package llm

import (
	"fmt"
	"net/http"
	"time"
)

const (
	defaultGrokBaseURL = "https://api.x.ai/v1"
	grokProviderName   = "xai-grok"
)

// GrokAdapter routes chat requests to the xAI Grok API using the
// OpenAI-compatible chat completions protocol.
type GrokAdapter struct {
	baseURL string
	apiKey  string
}

// NewGrokAdapter creates an adapter for the xAI Grok API.
// If baseURL is empty, it defaults to https://api.x.ai/v1.
func NewGrokAdapter(baseURL, apiKey string) *GrokAdapter {
	if baseURL == "" {
		baseURL = defaultGrokBaseURL
	}
	return &GrokAdapter{baseURL: baseURL, apiKey: apiKey}
}

// Client returns an HTTP client configured for xAI Grok.
func (a *GrokAdapter) Client() Client {
	return NewHTTPClientWithProvider(a.baseURL, a.apiKey, grokProviderName)
}

// Status probes the xAI Grok API and returns a typed status.
func (a *GrokAdapter) Status() ProviderStatus {
	if !a.reachable() {
		return ProviderStatus{
			Provider: grokProviderName,
			Runtime:  "chat_completions",
			Capabilities: ProviderCapabilities{
				PromptCache:     unavailableCapability("grok_unreachable"),
				ReasoningEcho:   unavailableCapability("grok_unreachable"),
				RateGuard:       unavailableCapability("grok_unreachable"),
				BudgetTelemetry: unavailableCapability("grok_unreachable"),
			},
		}
	}
	return openAICompatibleProviderStatus(grokProviderName, a.baseURL)
}

// reachable performs a lightweight health check against the xAI API.
func (a *GrokAdapter) reachable() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, a.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ClassifyGrokError maps xAI Grok error responses to the shared provider-error taxonomy.
func ClassifyGrokError(statusCode int, body string) ProviderErrorClassification {
	switch statusCode {
	case http.StatusTooManyRequests:
		return providerError(ProviderErrorRateLimit, ClassRetryable, statusCode, "xAI rate limit exceeded", true)
	case http.StatusUnauthorized:
		return providerError(ProviderErrorAuth, ClassFatal, statusCode, "xAI authentication failed", false)
	case http.StatusPaymentRequired:
		return providerError(ProviderErrorBilling, ClassFatal, statusCode, "xAI quota exceeded", false)
	case http.StatusServiceUnavailable:
		return providerError(ProviderErrorOverloaded, ClassRetryable, statusCode, "xAI service unavailable", true)
	case http.StatusInternalServerError:
		return providerError(ProviderErrorServerError, ClassRetryable, statusCode, "xAI internal server error", true)
	default:
		return providerError(ProviderErrorUnknown, ClassUnknown, statusCode, fmt.Sprintf("xAI error: HTTP %d", statusCode), false)
	}
}
