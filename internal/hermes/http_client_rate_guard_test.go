package hermes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClient_RateGuardBlocksRequestsWhenGenuineQuota(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining-Requests-1h", "0")
		w.Header().Set("X-RateLimit-Reset-Requests-1h", "1800")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer ts.Close()

	client := NewHTTPClientWithProvider(ts.URL, "key", "test").(*httpClient)

	_, err := client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if err == nil {
		t.Fatal("expected error on 429, got nil")
	}

	// Second request should be blocked by rate guard without hitting the server
	_, err = client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if err == nil {
		t.Fatal("expected rate guard to block second request, got nil")
	}
	if !strings.Contains(err.Error(), "rate_guard_active") {
		t.Fatalf("expected rate_guard_active error, got: %v", err)
	}

	status := client.ProviderStatus()
	if !status.Capabilities.RateGuard.Available {
		t.Fatal("expected RateGuard.Available=true")
	}
	if status.Capabilities.RateGuard.Reason != string(StatusNousRateLimited) {
		t.Fatalf("expected RateGuard.Reason=%q, got %q", StatusNousRateLimited, status.Capabilities.RateGuard.Reason)
	}
}

func TestHTTPClient_RateGuardAllowsRequestsAfterCooldown(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n"))
	}))
	defer ts.Close()

	client := NewHTTPClientWithProvider(ts.URL, "key", "test").(*httpClient)
	client.rateGuard = GuardState{
		LastKnownClass: RateLimitGenuineQuota,
		LastKnownAt:    time.Now().Add(-10 * time.Minute),
	}

	_, err := client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if err != nil {
		t.Fatalf("expected request to succeed after cooldown, got: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 server call, got %d", callCount)
	}
}

func TestHTTPClient_RateGuardDoesNotBlockOnUpstreamCapacity(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("X-RateLimit-Remaining-Requests", "198")
		w.Header().Set("X-RateLimit-Reset-Requests", "40")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer ts.Close()

	client := NewHTTPClientWithProvider(ts.URL, "key", "test").(*httpClient)

	_, err := client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if err == nil {
		t.Fatal("expected error on 429, got nil")
	}

	// Upstream capacity should not block subsequent requests
	_, err = client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if err == nil {
		t.Fatal("expected error on second 429, got nil")
	}
	if callCount != 2 {
		t.Fatalf("expected 2 server calls for upstream_capacity, got %d", callCount)
	}
}

func TestHTTPClient_OnCredentialExhaustedFiresOn429(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining-Requests-1h", "0")
		w.Header().Set("X-RateLimit-Reset-Requests-1h", "1800")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer ts.Close()

	client := NewHTTPClientWithProvider(ts.URL, "key", "test").(*httpClient)
	exhausted := false
	client.onCredentialExhausted = func(statusCode int, reason string, _ http.Header) {
		exhausted = true
		if statusCode != http.StatusTooManyRequests {
			t.Errorf("statusCode = %d, want %d", statusCode, http.StatusTooManyRequests)
		}
	}

	_, _ = client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if !exhausted {
		t.Fatal("expected exhaustion callback to fire on 429")
	}
}

func TestHTTPClient_OnCredentialExhaustedFiresOn401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer ts.Close()

	client := NewHTTPClientWithProvider(ts.URL, "key", "test").(*httpClient)
	exhausted := false
	client.onCredentialExhausted = func(statusCode int, reason string, _ http.Header) {
		exhausted = true
		if statusCode != http.StatusUnauthorized {
			t.Errorf("statusCode = %d, want %d", statusCode, http.StatusUnauthorized)
		}
		if reason != "auth_unauthorized" {
			t.Errorf("reason = %q, want auth_unauthorized", reason)
		}
	}

	_, _ = client.OpenStream(context.Background(), ChatRequest{Model: "gpt-4"})
	if !exhausted {
		t.Fatal("expected exhaustion callback to fire on 401")
	}
}

func TestSetOnCredentialExhausted_NoopOnNonHTTPClient(t *testing.T) {
	SetOnCredentialExhausted(nil, func(int, string, http.Header) {})
}
