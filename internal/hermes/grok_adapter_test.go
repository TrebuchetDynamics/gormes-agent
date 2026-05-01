package hermes

import (
	"net/http"
	"testing"
)

func TestGrokAdapter_Client(t *testing.T) {
	adapter := NewGrokAdapter("", "test-key")
	client := adapter.Client()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestGrokAdapter_StatusUnreachable(t *testing.T) {
	adapter := NewGrokAdapter("http://localhost:99999", "")
	status := adapter.Status()
	if status.Provider != grokProviderName {
		t.Fatalf("expected provider=%s, got %s", grokProviderName, status.Provider)
	}
	if status.Runtime != "chat_completions" {
		t.Fatalf("expected runtime=chat_completions, got %s", status.Runtime)
	}
	if status.Capabilities.PromptCache.Available {
		t.Fatal("expected prompt cache unavailable for unreachable server")
	}
}

func TestClassifyGrokError_RateLimit(t *testing.T) {
	result := ClassifyGrokError(http.StatusTooManyRequests, "")
	if result.Kind != ProviderErrorRateLimit {
		t.Fatalf("expected rate_limit, got %s", result.Kind)
	}
	if !result.Retryable {
		t.Fatal("expected rate limit to be retryable")
	}
}

func TestClassifyGrokError_Auth(t *testing.T) {
	result := ClassifyGrokError(http.StatusUnauthorized, "")
	if result.Kind != ProviderErrorAuth {
		t.Fatalf("expected auth, got %s", result.Kind)
	}
	if result.Retryable {
		t.Fatal("expected auth error to not be retryable")
	}
}

func TestClassifyGrokError_Overloaded(t *testing.T) {
	result := ClassifyGrokError(http.StatusServiceUnavailable, "")
	if result.Kind != ProviderErrorOverloaded {
		t.Fatalf("expected overloaded, got %s", result.Kind)
	}
	if !result.Retryable {
		t.Fatal("expected overloaded error to be retryable")
	}
}

func TestClassifyGrokError_ServerError(t *testing.T) {
	result := ClassifyGrokError(http.StatusInternalServerError, "")
	if result.Kind != ProviderErrorServerError {
		t.Fatalf("expected server_error, got %s", result.Kind)
	}
	if !result.Retryable {
		t.Fatal("expected server error to be retryable")
	}
}

func TestClassifyGrokError_Unknown(t *testing.T) {
	result := ClassifyGrokError(http.StatusBadRequest, "")
	if result.Kind != ProviderErrorUnknown {
		t.Fatalf("expected unknown, got %s", result.Kind)
	}
	if result.Retryable {
		t.Fatal("expected unknown error to not be retryable")
	}
}

func TestGrokAdapter_DefaultBaseURL(t *testing.T) {
	adapter := NewGrokAdapter("", "")
	if adapter.baseURL != defaultGrokBaseURL {
		t.Fatalf("expected default base URL, got %s", adapter.baseURL)
	}
}

func TestGrokAdapter_CustomBaseURL(t *testing.T) {
	adapter := NewGrokAdapter("https://custom.x.ai/v1", "key")
	if adapter.baseURL != "https://custom.x.ai/v1" {
		t.Fatalf("expected custom base URL, got %s", adapter.baseURL)
	}
	if adapter.apiKey != "key" {
		t.Fatal("expected api key to be set")
	}
}
