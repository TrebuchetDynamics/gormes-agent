package accountusage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AccountUsageReason string

const (
	AccountUsageReasonUnsupportedProvider AccountUsageReason = "unsupported_provider"
	AccountUsageReasonCredentialMissing   AccountUsageReason = "credential_missing"
	AccountUsageReasonOAuthRequired       AccountUsageReason = "oauth_required"
	AccountUsageReasonHTTPStatus          AccountUsageReason = "http_status"
	AccountUsageReasonMalformedPayload    AccountUsageReason = "malformed_payload"
	AccountUsageReasonRequestFailed       AccountUsageReason = "request_failed"
)

type AccountUsageFetchRequest struct {
	Provider  string
	BaseURL   string
	APIKey    string
	AccountID string
}

type AccountUsageHTTPRequest struct {
	URL     string
	Headers map[string]string
}

type AccountUsageHTTPResponse struct {
	StatusCode int
	Body       []byte
}

type AccountUsageHTTPClient interface {
	DoAccountUsageRequest(context.Context, AccountUsageHTTPRequest) (AccountUsageHTTPResponse, error)
}

type AccountUsageFetcher struct {
	client AccountUsageHTTPClient
	now    func() time.Time
}

func NewAccountUsageFetcher(client AccountUsageHTTPClient, now func() time.Time) AccountUsageFetcher {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return AccountUsageFetcher{client: client, now: now}
}

type AccountUsageSnapshot struct {
	Provider    string                   `json:"provider"`
	AccountID   string                   `json:"account_id,omitempty"`
	Plan        string                   `json:"plan,omitempty"`
	Source      string                   `json:"source,omitempty"`
	FetchedAt   time.Time                `json:"fetched_at"`
	Windows     []AccountUsageWindow     `json:"windows,omitempty"`
	Details     []string                 `json:"details,omitempty"`
	Unavailable *AccountUsageUnavailable `json:"unavailable,omitempty"`
}

func (s AccountUsageSnapshot) Available() bool {
	return s.Unavailable == nil && (len(s.Windows) > 0 || len(s.Details) > 0 || s.Plan != "")
}

type AccountUsageWindow struct {
	Label       string     `json:"label"`
	UsedPercent *float64   `json:"used_percent,omitempty"`
	ResetAt     *time.Time `json:"reset_at,omitempty"`
	Limit       *float64   `json:"limit,omitempty"`
	Remaining   *float64   `json:"remaining,omitempty"`
	Used        *float64   `json:"used,omitempty"`
}

type AccountUsageUnavailable struct {
	Reason     AccountUsageReason `json:"reason"`
	Message    string             `json:"message"`
	Endpoint   string             `json:"endpoint,omitempty"`
	StatusCode int                `json:"status_code,omitempty"`
}

type AccountUsageRenderOptions struct {
	IncludeDetails bool
}

func (f AccountUsageFetcher) Fetch(ctx context.Context, req AccountUsageFetchRequest) (AccountUsageSnapshot, error) {
	provider := normalizeAccountUsageProvider(req.Provider)
	base := AccountUsageSnapshot{
		Provider:  provider,
		AccountID: strings.TrimSpace(req.AccountID),
		FetchedAt: f.now().UTC(),
	}
	switch provider {
	case "openai-codex":
		return f.fetchCodex(ctx, base, req), nil
	case "anthropic":
		return f.fetchAnthropic(ctx, base, req), nil
	case "openrouter":
		return f.fetchOpenRouter(ctx, base, req), nil
	default:
		base.Unavailable = &AccountUsageUnavailable{
			Reason:  AccountUsageReasonUnsupportedProvider,
			Message: "account usage is not supported for provider " + provider,
		}
		return base, nil
	}
}

func (f AccountUsageFetcher) fetchCodex(ctx context.Context, base AccountUsageSnapshot, req AccountUsageFetchRequest) AccountUsageSnapshot {
	if strings.TrimSpace(req.APIKey) == "" {
		base.Unavailable = credentialMissing("openai-codex")
		return base
	}
	endpoint := codexUsageEndpoint(req.BaseURL)
	resp, err := f.do(ctx, endpoint, map[string]string{
		"Authorization":      "Bearer " + req.APIKey,
		"ChatGPT-Account-Id": strings.TrimSpace(req.AccountID),
	})
	if err != nil {
		base.Unavailable = requestFailed(endpoint, err)
		return base
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		base.Unavailable = httpStatus(endpoint, resp.StatusCode)
		return base
	}
	var payload struct {
		PlanType  string `json:"plan_type"`
		RateLimit struct {
			PrimaryWindow   accountUsageRawWindow `json:"primary_window"`
			SecondaryWindow accountUsageRawWindow `json:"secondary_window"`
		} `json:"rate_limit"`
		Credits struct {
			HasCredits bool    `json:"has_credits"`
			Balance    float64 `json:"balance"`
		} `json:"credits"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		base.Unavailable = malformedPayload(endpoint)
		return base
	}
	base.Plan = titleAccountUsagePlan(payload.PlanType)
	base.Source = "codex_usage_api"
	base.Windows = append(base.Windows, accountUsageWindow("Session", payload.RateLimit.PrimaryWindow, false))
	base.Windows = append(base.Windows, accountUsageWindow("Weekly", payload.RateLimit.SecondaryWindow, false))
	base.Windows = compactAccountUsageWindows(base.Windows)
	if payload.Credits.HasCredits {
		base.Details = append(base.Details, fmt.Sprintf("Credits balance: $%.2f", payload.Credits.Balance))
	}
	return base
}

func (f AccountUsageFetcher) fetchAnthropic(ctx context.Context, base AccountUsageSnapshot, req AccountUsageFetchRequest) AccountUsageSnapshot {
	key := strings.TrimSpace(req.APIKey)
	if key == "" {
		base.Unavailable = credentialMissing("anthropic")
		return base
	}
	if strings.HasPrefix(key, "sk-ant") {
		base.Unavailable = &AccountUsageUnavailable{
			Reason:  AccountUsageReasonOAuthRequired,
			Message: "Anthropic account usage requires OAuth credentials",
		}
		return base
	}
	endpoint := anthropicUsageEndpoint(req.BaseURL)
	resp, err := f.do(ctx, endpoint, map[string]string{"Authorization": "Bearer " + key})
	if err != nil {
		base.Unavailable = requestFailed(endpoint, err)
		return base
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		base.Unavailable = httpStatus(endpoint, resp.StatusCode)
		return base
	}
	var payload struct {
		FiveHour   accountUsageRawWindow `json:"five_hour"`
		SevenDay   accountUsageRawWindow `json:"seven_day"`
		ExtraUsage struct {
			IsEnabled    bool    `json:"is_enabled"`
			UsedCredits  float64 `json:"used_credits"`
			MonthlyLimit float64 `json:"monthly_limit"`
			Currency     string  `json:"currency"`
		} `json:"extra_usage"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		base.Unavailable = malformedPayload(endpoint)
		return base
	}
	base.Source = "anthropic_oauth_usage_api"
	base.Windows = compactAccountUsageWindows([]AccountUsageWindow{
		accountUsageWindow("5-hour", payload.FiveHour, true),
		accountUsageWindow("7-day", payload.SevenDay, true),
	})
	if payload.ExtraUsage.IsEnabled {
		currency := strings.TrimSpace(payload.ExtraUsage.Currency)
		if currency == "" {
			currency = "USD"
		}
		base.Details = append(base.Details, fmt.Sprintf("Extra usage: %.2f / %.2f %s", payload.ExtraUsage.UsedCredits, payload.ExtraUsage.MonthlyLimit, currency))
	}
	return base
}

func (f AccountUsageFetcher) fetchOpenRouter(ctx context.Context, base AccountUsageSnapshot, req AccountUsageFetchRequest) AccountUsageSnapshot {
	key := strings.TrimSpace(req.APIKey)
	if key == "" {
		base.Unavailable = credentialMissing("openrouter")
		return base
	}
	baseURL := cleanAccountUsageBaseURL(firstNonEmptyString(strings.TrimSpace(req.BaseURL), "https://openrouter.ai/api/v1"))
	creditsEndpoint := baseURL + "/credits"
	keyEndpoint := baseURL + "/key"
	headers := map[string]string{"Authorization": "Bearer " + key}

	creditsResp, err := f.do(ctx, creditsEndpoint, headers)
	if err != nil {
		base.Unavailable = requestFailed(creditsEndpoint, err)
		return base
	}
	if creditsResp.StatusCode < 200 || creditsResp.StatusCode >= 300 {
		base.Unavailable = httpStatus(creditsEndpoint, creditsResp.StatusCode)
		return base
	}
	var creditsPayload struct {
		Data struct {
			TotalCredits float64 `json:"total_credits"`
			TotalUsage   float64 `json:"total_usage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(creditsResp.Body, &creditsPayload); err != nil {
		base.Unavailable = malformedPayload(creditsEndpoint)
		return base
	}

	keyResp, err := f.do(ctx, keyEndpoint, headers)
	if err != nil {
		base.Unavailable = requestFailed(keyEndpoint, err)
		return base
	}
	if keyResp.StatusCode < 200 || keyResp.StatusCode >= 300 {
		base.Unavailable = httpStatus(keyEndpoint, keyResp.StatusCode)
		return base
	}
	var keyPayload struct {
		Data struct {
			Limit          float64 `json:"limit"`
			LimitRemaining float64 `json:"limit_remaining"`
			LimitReset     string  `json:"limit_reset"`
			Usage          float64 `json:"usage"`
			UsageDaily     float64 `json:"usage_daily"`
			UsageWeekly    float64 `json:"usage_weekly"`
			UsageMonthly   float64 `json:"usage_monthly"`
		} `json:"data"`
	}
	if err := json.Unmarshal(keyResp.Body, &keyPayload); err != nil {
		base.Unavailable = malformedPayload(keyEndpoint)
		return base
	}

	base.Source = "openrouter_usage_api"
	if keyPayload.Data.Limit > 0 {
		used := keyPayload.Data.Limit - keyPayload.Data.LimitRemaining
		usedPercent := roundAccountUsagePercent(used / keyPayload.Data.Limit * 100)
		limit := keyPayload.Data.Limit
		remaining := keyPayload.Data.LimitRemaining
		base.Windows = append(base.Windows, AccountUsageWindow{
			Label:       "API key quota",
			UsedPercent: &usedPercent,
			Limit:       &limit,
			Remaining:   &remaining,
			Used:        &used,
		})
	}
	balance := creditsPayload.Data.TotalCredits - creditsPayload.Data.TotalUsage
	base.Details = append(base.Details, fmt.Sprintf("Credits balance: $%.2f", balance))
	base.Details = append(base.Details, fmt.Sprintf("API key usage: $%.2f total - $%.2f today - $%.2f this week - $%.2f this month",
		keyPayload.Data.Usage,
		keyPayload.Data.UsageDaily,
		keyPayload.Data.UsageWeekly,
		keyPayload.Data.UsageMonthly,
	))
	return base
}

func (f AccountUsageFetcher) do(ctx context.Context, endpoint string, headers map[string]string) (AccountUsageHTTPResponse, error) {
	if f.client == nil {
		return AccountUsageHTTPResponse{}, fmt.Errorf("account usage HTTP client is required")
	}
	copiedHeaders := make(map[string]string, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			copiedHeaders[key] = value
		}
	}
	return f.client.DoAccountUsageRequest(ctx, AccountUsageHTTPRequest{
		URL:     endpoint,
		Headers: copiedHeaders,
	})
}

func RenderAccountUsageLines(snapshot AccountUsageSnapshot, _ AccountUsageRenderOptions) []string {
	provider := snapshot.Provider
	if provider == "" {
		provider = "unknown"
	}
	label := provider
	if snapshot.Plan != "" {
		label += " (" + snapshot.Plan + ")"
	}
	lines := []string{"Provider: " + label}
	if snapshot.Unavailable != nil {
		lines = append(lines, "Usage unavailable: "+snapshot.Unavailable.Message)
		return lines
	}
	for _, window := range snapshot.Windows {
		lines = append(lines, renderAccountUsageWindow(window))
	}
	lines = append(lines, snapshot.Details...)
	return lines
}

func RenderAccountUsageJSON(snapshot AccountUsageSnapshot) ([]byte, error) {
	snapshot = redactAccountUsageSnapshot(snapshot)
	return json.MarshalIndent(snapshot, "", "  ")
}

type accountUsageRawWindow struct {
	UsedPercent json.RawMessage `json:"used_percent"`
	Utilization json.RawMessage `json:"utilization"`
	ResetAt     json.RawMessage `json:"reset_at"`
	ResetsAt    json.RawMessage `json:"resets_at"`
}

func accountUsageWindow(label string, raw accountUsageRawWindow, utilizationFraction bool) AccountUsageWindow {
	var window AccountUsageWindow
	window.Label = label
	if used, ok := parseAccountUsageFloat(firstRaw(raw.UsedPercent, raw.Utilization)); ok {
		if utilizationFraction && used <= 1 {
			used *= 100
		}
		used = roundAccountUsagePercent(used)
		window.UsedPercent = &used
	}
	if reset, ok := parseAccountUsageTime(firstRaw(raw.ResetAt, raw.ResetsAt)); ok {
		window.ResetAt = &reset
	}
	return window
}

func compactAccountUsageWindows(in []AccountUsageWindow) []AccountUsageWindow {
	out := make([]AccountUsageWindow, 0, len(in))
	for _, window := range in {
		if window.UsedPercent != nil || window.ResetAt != nil || window.Limit != nil || window.Remaining != nil || window.Used != nil {
			out = append(out, window)
		}
	}
	return out
}

func renderAccountUsageWindow(window AccountUsageWindow) string {
	if window.UsedPercent == nil {
		return window.Label + ": usage unavailable"
	}
	used := *window.UsedPercent
	remaining := math.Max(0, 100-used)
	return fmt.Sprintf("%s: %.0f%% remaining (%.0f%% used)", window.Label, remaining, used)
}

func parseAccountUsageFloat(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var num float64
	if err := json.Unmarshal(raw, &num); err == nil {
		return num, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, false
	}
	num, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	return num, err == nil
}

func parseAccountUsageTime(raw json.RawMessage) (time.Time, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, false
	}
	var unix float64
	if err := json.Unmarshal(raw, &unix); err == nil {
		return time.Unix(int64(unix), 0).UTC(), true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return time.Time{}, false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return time.Time{}, false
	}
	if unixInt, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return time.Unix(unixInt, 0).UTC(), true
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 && string(value) != "null" {
			return value
		}
	}
	return nil
}

func normalizeAccountUsageProvider(provider string) string {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	switch normalized {
	case "codex", "openai", "openai-codex":
		return "openai-codex"
	case "anthropic", "claude":
		return "anthropic"
	case "openrouter":
		return "openrouter"
	default:
		if normalized == "" {
			return "unknown"
		}
		return normalized
	}
}

func codexUsageEndpoint(baseURL string) string {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		return "https://chatgpt.com/backend-api/wham/usage"
	}
	if strings.Contains(base, "/codex") {
		return strings.Replace(base, "/codex", "/wham/usage", 1)
	}
	return strings.TrimRight(base, "/") + "/wham/usage"
}

func anthropicUsageEndpoint(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "https://api.anthropic.com/api/oauth/usage"
	}
	return base + "/api/oauth/usage"
}

func cleanAccountUsageBaseURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func titleAccountUsagePlan(plan string) string {
	trimmed := strings.TrimSpace(plan)
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed[:1]) + strings.ToLower(trimmed[1:])
}

func credentialMissing(provider string) *AccountUsageUnavailable {
	return &AccountUsageUnavailable{
		Reason:  AccountUsageReasonCredentialMissing,
		Message: "account usage credentials are missing for " + provider,
	}
}

func requestFailed(endpoint string, err error) *AccountUsageUnavailable {
	return &AccountUsageUnavailable{
		Reason:   AccountUsageReasonRequestFailed,
		Message:  "account usage request failed: " + err.Error(),
		Endpoint: redactAccountUsageEndpoint(endpoint),
	}
}

func httpStatus(endpoint string, status int) *AccountUsageUnavailable {
	return &AccountUsageUnavailable{
		Reason:     AccountUsageReasonHTTPStatus,
		Message:    fmt.Sprintf("provider returned status %d", status),
		Endpoint:   redactAccountUsageEndpoint(endpoint),
		StatusCode: status,
	}
}

func malformedPayload(endpoint string) *AccountUsageUnavailable {
	return &AccountUsageUnavailable{
		Reason:   AccountUsageReasonMalformedPayload,
		Message:  "provider returned malformed account usage payload",
		Endpoint: redactAccountUsageEndpoint(endpoint),
	}
}

func redactAccountUsageSnapshot(snapshot AccountUsageSnapshot) AccountUsageSnapshot {
	if snapshot.Unavailable != nil {
		unavailable := *snapshot.Unavailable
		unavailable.Endpoint = redactAccountUsageEndpoint(unavailable.Endpoint)
		snapshot.Unavailable = &unavailable
	}
	return snapshot
}

func redactAccountUsageEndpoint(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.RawQuery == "" {
		return trimmed
	}
	parsed.RawQuery = "<redacted>"
	return parsed.String()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func roundAccountUsagePercent(value float64) float64 {
	return math.Round(value*100) / 100
}
