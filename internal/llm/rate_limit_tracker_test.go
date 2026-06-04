package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRateLimitHeadersTracksNousStyleBuckets(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	headers := http.Header{}
	headers.Set("X-RateLimit-Limit-Requests", "200")
	headers.Set("X-RateLimit-Limit-Requests-1h", "1000")
	headers.Set("X-RateLimit-Limit-Tokens", "800000")
	headers.Set("X-RateLimit-Limit-Tokens-1h", "8000000")
	headers.Set("X-RateLimit-Remaining-Requests", "150")
	headers.Set("X-RateLimit-Remaining-Requests-1h", "720")
	headers.Set("X-RateLimit-Remaining-Tokens", "640000")
	headers.Set("X-RateLimit-Remaining-Tokens-1h", "7999856")
	headers.Set("X-RateLimit-Reset-Requests", "58")
	headers.Set("X-RateLimit-Reset-Requests-1h", "3700")
	headers.Set("X-RateLimit-Reset-Tokens", "134")
	headers.Set("X-RateLimit-Reset-Tokens-1h", "3600")

	state, ok := ParseRateLimitHeaders(headers, "openrouter", now)
	if !ok {
		t.Fatal("ParseRateLimitHeaders returned ok=false")
	}
	if !state.HasData() {
		t.Fatal("state.HasData() = false, want true")
	}
	if state.Provider != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", state.Provider)
	}
	if got := state.RequestsMinute.Used(); got != 50 {
		t.Fatalf("RequestsMinute.Used() = %d, want 50", got)
	}
	if got := state.RequestsMinute.UsagePercent(); got != 25 {
		t.Fatalf("RequestsMinute.UsagePercent() = %v, want 25", got)
	}
	if got := state.TokensHour.Remaining; got != 7_999_856 {
		t.Fatalf("TokensHour.Remaining = %d, want 7999856", got)
	}
	if got := state.RequestsHour.RemainingDuration(now.Add(100 * time.Second)); got != time.Hour {
		t.Fatalf("RequestsHour remaining reset = %s, want 1h", got)
	}
}

func TestFormatRateLimitDisplayMatchesHermesShape(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	state, ok := ParseRateLimitHeaders(http.Header{
		"x-ratelimit-limit-requests":        {"200"},
		"x-ratelimit-remaining-requests":    {"5"},
		"x-ratelimit-reset-requests":        {"58"},
		"x-ratelimit-limit-requests-1h":     {"1000"},
		"x-ratelimit-remaining-requests-1h": {"720"},
		"x-ratelimit-reset-requests-1h":     {"3700"},
		"x-ratelimit-limit-tokens":          {"800000"},
		"x-ratelimit-remaining-tokens":      {"640000"},
		"x-ratelimit-reset-tokens":          {"134"},
		"x-ratelimit-limit-tokens-1h":       {"8000000"},
		"x-ratelimit-remaining-tokens-1h":   {"7999856"},
		"x-ratelimit-reset-tokens-1h":       {"3600"},
	}, "nous", now)
	if !ok {
		t.Fatal("ParseRateLimitHeaders returned ok=false")
	}

	display := FormatRateLimitDisplayAt(state, now.Add(10*time.Second))
	for _, want := range []string{
		"Nous Rate Limits (captured 10s ago):",
		"Requests/min",
		"[███████████████████░]  97.5%  195/200 used  (5 left, resets in 48s)",
		"Requests/hr",
		"280/1.0K used  (720 left, resets in 1h 1m)",
		"Tokens/min",
		"160.0K/800.0K used  (640.0K left, resets in 2m 4s)",
		"Tokens/hr",
		"144/8.0M used  (8.0M left, resets in 59m 50s)",
		"⚠ requests/min at 98% — resets in 48s",
	} {
		if !strings.Contains(display, want) {
			t.Fatalf("display missing %q:\n%s", want, display)
		}
	}

	compact := FormatRateLimitCompactAt(state, now.Add(10*time.Second))
	for _, want := range []string{
		"RPM: 5/200",
		"RPH: 720/1.0K (resets 1h 1m)",
		"TPM: 640.0K/800.0K",
		"TPH: 8.0M/8.0M (resets 59m 50s)",
	} {
		if !strings.Contains(compact, want) {
			t.Fatalf("compact missing %q: %s", want, compact)
		}
	}
}

func TestHTTPClientProviderStatusCapturesRateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-RateLimit-Limit-Requests", "100")
		w.Header().Set("X-RateLimit-Remaining-Requests", "80")
		w.Header().Set("X-RateLimit-Reset-Requests", "30")
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"ok"}}]}`)
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"finish_reason":"stop"}]}`)
	}))
	defer server.Close()

	client := NewHTTPClientWithProvider(server.URL, "sk-test", "openrouter")
	stream, err := client.OpenStream(context.Background(), ChatRequest{Model: "test-model", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("OpenStream() err = %v", err)
	}
	defer stream.Close()

	status := ProviderStatusOf(client)
	if !status.RateLimit.HasData() {
		t.Fatalf("status.RateLimit.HasData() = false, want captured headers: %+v", status.RateLimit)
	}
	if status.RateLimit.Provider != "openrouter" {
		t.Fatalf("status.RateLimit.Provider = %q, want openrouter", status.RateLimit.Provider)
	}
	if got := status.RateLimit.RequestsMinute.Used(); got != 20 {
		t.Fatalf("RequestsMinute.Used() = %d, want 20", got)
	}
}

func TestParseRateLimitHeadersRequiresRateLimitHeaders(t *testing.T) {
	if state, ok := ParseRateLimitHeaders(http.Header{"Retry-After": {"12"}}, "openai", time.Now()); ok || state.HasData() {
		t.Fatalf("ParseRateLimitHeaders with no x-ratelimit headers = (%+v, %v), want no data", state, ok)
	}
}

func TestRateLimitMalformedBucketDegradesToNoDataBucket(t *testing.T) {
	state, ok := ParseRateLimitHeaders(http.Header{
		"X-RateLimit-Limit-Requests":     {"not-a-number"},
		"X-RateLimit-Remaining-Requests": {"also-bad"},
		"X-RateLimit-Reset-Requests":     {"bad"},
	}, "openai", time.Unix(0, 0))
	if !ok {
		t.Fatal("ParseRateLimitHeaders returned ok=false despite x-ratelimit header presence")
	}
	if state.RequestsMinute.Limit != 0 || state.RequestsMinute.Remaining != 0 || state.RequestsMinute.ResetAfter != 0 {
		t.Fatalf("malformed bucket = %+v, want zero no-data bucket", state.RequestsMinute)
	}
	line := FormatRateLimitDisplayAt(state, state.CapturedAt)
	if !strings.Contains(line, "Requests/min    (no data)") {
		t.Fatalf("display = %q, want no data request bucket", line)
	}
}
