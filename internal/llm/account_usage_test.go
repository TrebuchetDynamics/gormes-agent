package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAccountUsageCodexNormalizesWindowsAndCredits(t *testing.T) {
	client := &fakeAccountUsageHTTP{
		responses: map[string]fakeAccountUsageResponse{
			"https://chatgpt.com/backend-api/wham/usage": {
				status: 200,
				body: `{
					"plan_type": "pro",
					"rate_limit": {
						"primary_window": {"used_percent": 15, "reset_at": 1900000000},
						"secondary_window": {"used_percent": 40, "reset_at": "2030-03-17T19:40:00Z"}
					},
					"credits": {"has_credits": true, "balance": 12.5}
				}`,
			},
		},
	}
	fetcher := NewAccountUsageFetcher(client, fixedAccountUsageClock)

	got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{
		Provider:  "openai-codex",
		BaseURL:   "https://chatgpt.com/backend-api/codex",
		APIKey:    "sk-codex-secret",
		AccountID: "acct_123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !got.Available() || got.Provider != "openai-codex" || got.Plan != "Pro" {
		t.Fatalf("snapshot = %+v, want available Codex Pro", got)
	}
	if len(got.Windows) != 2 {
		t.Fatalf("windows len = %d, want 2", len(got.Windows))
	}
	if got.Windows[0].Label != "Session" || got.Windows[0].UsedPercent == nil || *got.Windows[0].UsedPercent != 15 {
		t.Fatalf("first window = %+v, want Session 15%% used", got.Windows[0])
	}
	if got.Windows[0].ResetAt == nil || got.Windows[0].ResetAt.Unix() != 1900000000 {
		t.Fatalf("first reset = %v, want unix 1900000000", got.Windows[0].ResetAt)
	}
	if got.Windows[1].Label != "Weekly" || got.Windows[1].ResetAt == nil || got.Windows[1].ResetAt.Year() != 2030 {
		t.Fatalf("second window = %+v, want Weekly parsed ISO reset", got.Windows[1])
	}
	if !hasAccountUsageDetail(got.Details, "Credits balance: $12.50") {
		t.Fatalf("details = %+v, want credit balance", got.Details)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(client.requests))
	}
	req := client.requests[0]
	if req.Headers["Authorization"] != "Bearer sk-codex-secret" || req.Headers["ChatGPT-Account-Id"] != "acct_123" {
		t.Fatalf("headers = %+v, want auth plus account id", req.Headers)
	}
	lines := RenderAccountUsageLines(got, AccountUsageRenderOptions{})
	if !hasAccountUsageLine(lines, "Provider: openai-codex (Pro)") || !hasAccountUsageLine(lines, "Session: 85% remaining (15% used)") {
		t.Fatalf("rendered lines = %+v", lines)
	}
}

func TestAccountUsageAnthropicOAuthAndCredentialMissing(t *testing.T) {
	t.Run("oauth_usage", func(t *testing.T) {
		client := &fakeAccountUsageHTTP{
			responses: map[string]fakeAccountUsageResponse{
				"https://api.anthropic.com/api/oauth/usage": {
					status: 200,
					body: `{
						"five_hour": {"utilization": 0.25, "resets_at": "2030-03-17T19:40:00Z"},
						"seven_day": {"utilization": 55, "resets_at": 1900000000},
						"extra_usage": {"is_enabled": true, "used_credits": 2.5, "monthly_limit": 20, "currency": "USD"}
					}`,
				},
			},
		}
		fetcher := NewAccountUsageFetcher(client, fixedAccountUsageClock)

		got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{
			Provider: "anthropic",
			APIKey:   "oauth-ant-secret",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !got.Available() || len(got.Windows) != 2 {
			t.Fatalf("snapshot = %+v, want available Anthropic windows", got)
		}
		if got.Windows[0].UsedPercent == nil || *got.Windows[0].UsedPercent != 25 {
			t.Fatalf("five_hour window = %+v, want utilization converted to 25%%", got.Windows[0])
		}
		if !hasAccountUsageDetail(got.Details, "Extra usage: 2.50 / 20.00 USD") {
			t.Fatalf("details = %+v, want extra usage", got.Details)
		}
		if got.Unavailable != nil {
			t.Fatalf("unavailable = %+v, want nil", got.Unavailable)
		}
	})

	t.Run("api_key_reports_nonfatal_unavailable", func(t *testing.T) {
		fetcher := NewAccountUsageFetcher(&fakeAccountUsageHTTP{}, fixedAccountUsageClock)
		got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{
			Provider: "anthropic",
			APIKey:   "sk-ant-api-key",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Available() || got.Unavailable == nil || got.Unavailable.Reason != AccountUsageReasonOAuthRequired {
			t.Fatalf("snapshot = %+v, want OAuth-required unavailable evidence", got)
		}
	})
}

func TestAccountUsageOpenRouterCreditsQuotaAndUsage(t *testing.T) {
	client := &fakeAccountUsageHTTP{
		responses: map[string]fakeAccountUsageResponse{
			"https://openrouter.ai/api/v1/credits": {
				status: 200,
				body:   `{"data":{"total_credits":300.0,"total_usage":10.92}}`,
			},
			"https://openrouter.ai/api/v1/key": {
				status: 200,
				body: `{"data":{
					"limit":100.0,
					"limit_remaining":70.0,
					"limit_reset":"monthly",
					"usage":12.5,
					"usage_daily":0.5,
					"usage_weekly":2.0,
					"usage_monthly":8.0,
					"rate_limit":{"requests":-1,"interval":"10s"}
				}}`,
			},
		},
	}
	fetcher := NewAccountUsageFetcher(client, fixedAccountUsageClock)

	got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{
		Provider: "openrouter",
		BaseURL:  "https://openrouter.ai/api/v1",
		APIKey:   "sk-openrouter-secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !got.Available() || len(got.Windows) != 1 {
		t.Fatalf("snapshot = %+v, want one quota window", got)
	}
	if got.Windows[0].Label != "API key quota" || got.Windows[0].UsedPercent == nil || *got.Windows[0].UsedPercent != 30 {
		t.Fatalf("quota window = %+v, want 30%% used", got.Windows[0])
	}
	if !hasAccountUsageDetail(got.Details, "Credits balance: $289.08") {
		t.Fatalf("details = %+v, want credits balance", got.Details)
	}
	if !hasAccountUsageDetail(got.Details, "API key usage: $12.50 total - $0.50 today - $2.00 this week - $8.00 this month") {
		t.Fatalf("details = %+v, want API key usage", got.Details)
	}
	for _, line := range RenderAccountUsageLines(got, AccountUsageRenderOptions{}) {
		if strings.Contains(line, "-1 requests / 10s") {
			t.Fatalf("rendered deprecated OpenRouter rate limit: %q", line)
		}
		if strings.Contains(line, "sk-openrouter-secret") {
			t.Fatalf("render leaked API key: %q", line)
		}
	}
}

func TestAccountUsageUnsupportedMissingAndDegradedEvidence(t *testing.T) {
	t.Run("unsupported_provider", func(t *testing.T) {
		fetcher := NewAccountUsageFetcher(&fakeAccountUsageHTTP{}, fixedAccountUsageClock)
		got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{Provider: "custom"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Available() || got.Unavailable == nil || got.Unavailable.Reason != AccountUsageReasonUnsupportedProvider {
			t.Fatalf("snapshot = %+v, want unsupported provider evidence", got)
		}
	})

	t.Run("missing_credentials", func(t *testing.T) {
		fetcher := NewAccountUsageFetcher(&fakeAccountUsageHTTP{}, fixedAccountUsageClock)
		got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{Provider: "openrouter"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Available() || got.Unavailable == nil || got.Unavailable.Reason != AccountUsageReasonCredentialMissing {
			t.Fatalf("snapshot = %+v, want missing credential evidence", got)
		}
	})

	t.Run("http_failure_redacts_endpoint", func(t *testing.T) {
		client := &fakeAccountUsageHTTP{
			responses: map[string]fakeAccountUsageResponse{
				"https://openrouter.ai/api/v1/credits": {
					status: 429,
					body:   `{"error":"slow down"}`,
				},
			},
		}
		fetcher := NewAccountUsageFetcher(client, fixedAccountUsageClock)
		got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{
			Provider: "openrouter",
			BaseURL:  "https://openrouter.ai/api/v1?api_key=secret",
			APIKey:   "sk-openrouter-secret",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Available() || got.Unavailable == nil || got.Unavailable.Reason != AccountUsageReasonHTTPStatus {
			t.Fatalf("snapshot = %+v, want HTTP degraded evidence", got)
		}
		if got.Unavailable.StatusCode != 429 || strings.Contains(got.Unavailable.Endpoint, "secret") {
			t.Fatalf("unavailable = %+v, want 429 and redacted endpoint", got.Unavailable)
		}
	})

	t.Run("malformed_json_is_degraded", func(t *testing.T) {
		client := &fakeAccountUsageHTTP{
			responses: map[string]fakeAccountUsageResponse{
				"https://chatgpt.com/backend-api/wham/usage": {status: 200, body: `not-json`},
			},
		}
		fetcher := NewAccountUsageFetcher(client, fixedAccountUsageClock)
		got, err := fetcher.Fetch(context.Background(), AccountUsageFetchRequest{
			Provider: "openai-codex",
			APIKey:   "sk-codex-secret",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Available() || got.Unavailable == nil || got.Unavailable.Reason != AccountUsageReasonMalformedPayload {
			t.Fatalf("snapshot = %+v, want malformed-payload evidence", got)
		}
	})
}

func TestAccountUsageRenderJSONRedactsEvidence(t *testing.T) {
	snapshot := AccountUsageSnapshot{
		Provider:  "openrouter",
		Source:    "credits_api",
		FetchedAt: fixedAccountUsageClock(),
		Unavailable: &AccountUsageUnavailable{
			Reason:     AccountUsageReasonHTTPStatus,
			Message:    "provider returned status 401",
			Endpoint:   "https://openrouter.ai/api/v1/key?<redacted>",
			StatusCode: 401,
		},
	}
	data, err := RenderAccountUsageJSON(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret") {
		t.Fatalf("json leaked secret: %s", data)
	}
	var decoded AccountUsageSnapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Unavailable == nil || decoded.Unavailable.Endpoint != "https://openrouter.ai/api/v1/key?<redacted>" {
		t.Fatalf("decoded = %+v, want unavailable redacted endpoint", decoded)
	}
}

func fixedAccountUsageClock() time.Time {
	return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
}

type fakeAccountUsageResponse struct {
	status int
	body   string
	err    error
}

type fakeAccountUsageHTTP struct {
	responses map[string]fakeAccountUsageResponse
	requests  []AccountUsageHTTPRequest
}

func (c *fakeAccountUsageHTTP) DoAccountUsageRequest(_ context.Context, req AccountUsageHTTPRequest) (AccountUsageHTTPResponse, error) {
	c.requests = append(c.requests, req)
	if response, ok := c.responses[req.URL]; ok {
		if response.err != nil {
			return AccountUsageHTTPResponse{}, response.err
		}
		return AccountUsageHTTPResponse{
			StatusCode: response.status,
			Body:       []byte(response.body),
		}, nil
	}
	return AccountUsageHTTPResponse{}, errors.New("unexpected URL " + req.URL)
}

func hasAccountUsageDetail(details []string, want string) bool {
	for _, detail := range details {
		if detail == want {
			return true
		}
	}
	return false
}

func hasAccountUsageLine(lines []string, want string) bool {
	for _, line := range lines {
		if strings.Contains(line, want) {
			return true
		}
	}
	return false
}
