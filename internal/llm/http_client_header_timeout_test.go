package llm

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPClientWithProvider_UsesProviderFriendlyHeaderTimeout(t *testing.T) {
	client := NewHTTPClientWithProvider("https://openrouter.ai/api/v1", "test-key", "openrouter")
	hc, ok := client.(*httpClient)
	if !ok {
		t.Fatalf("NewHTTPClientWithProvider returned %T, want *httpClient", client)
	}
	transport, ok := hc.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("http transport = %T, want *http.Transport", hc.http.Transport)
	}
	if transport.ResponseHeaderTimeout < 30*time.Second {
		t.Fatalf("ResponseHeaderTimeout = %v, want at least 30s so slow OpenRouter routing does not fail before first token", transport.ResponseHeaderTimeout)
	}
	if hc.http.Timeout != 0 {
		t.Fatalf("http.Client Timeout = %v, want 0 so streaming body reads are not globally capped", hc.http.Timeout)
	}
}
