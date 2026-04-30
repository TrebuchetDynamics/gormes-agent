package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	defaultBrowserbaseBaseURL = "https://api.browserbase.com"
	defaultFirecrawlBaseURL   = "https://api.firecrawl.dev"
	defaultFirecrawlTTL       = 300

	BrowserProviderEvidenceUnconfigured            = "browser_provider_unconfigured"
	BrowserProviderEvidenceSessionCreated          = "browser_provider_session_created"
	BrowserProviderEvidenceBrowserbasePlanFallback = "browserbase_plan_fallback"
	BrowserProviderEvidenceCreateFailed            = "browser_provider_create_failed"
	BrowserProviderEvidenceCleanupOK               = "browser_provider_cleanup_ok"
	BrowserProviderEvidenceCleanupSkipped          = "browser_provider_cleanup_skipped"
	BrowserProviderEvidenceCleanupFailed           = "browser_provider_cleanup_failed"
	BrowserProviderEvidenceProviderErrorRedacted   = "browser_provider_error_redacted"
	BrowserProviderFeatureBasicStealth             = "basic_stealth"
	BrowserProviderFeatureProxies                  = "proxies"
	BrowserProviderFeatureAdvancedStealth          = "advanced_stealth"
	BrowserProviderFeatureKeepAlive                = "keep_alive"
	BrowserProviderFeatureCustomTimeout            = "custom_timeout"
	BrowserProviderFeatureFirecrawl                = "firecrawl"
)

// BrowserProviderHTTPClient is satisfied by *http.Client and hermetic tests.
type BrowserProviderHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// BrowserbaseProviderConfig is the direct Browserbase credential and option
// shape. It mirrors Hermes' env-driven provider without reading env at
// construction time unless callers explicitly use BrowserbaseProviderConfigFromEnv.
type BrowserbaseProviderConfig struct {
	APIKey             string
	ProjectID          string
	BaseURL            string
	EnableProxies      bool
	EnableKeepAlive    bool
	AdvancedStealth    bool
	SessionTimeoutMS   int
	UseExplicitOptions bool
}

// FirecrawlProviderConfig is the direct Firecrawl browser credential and
// request-default shape.
type FirecrawlProviderConfig struct {
	APIKey     string
	BaseURL    string
	BrowserTTL int
}

// BrowserProviderSessionRequest identifies one task-scoped browser session.
type BrowserProviderSessionRequest struct {
	TaskID string
}

// BrowserProviderSession is the provider-neutral session envelope consumed by
// later browser runtimes. CompatibilitySessionID preserves Hermes' bb_session_id
// behavior even when the provider is not Browserbase.
type BrowserProviderSession struct {
	ProviderName           string
	SessionName            string
	ProviderSessionID      string
	CompatibilitySessionID string
	CDPURL                 string
	Features               map[string]bool
	Evidence               []string
	Redacted               bool
}

// BrowserProviderCleanupResult is a redacted close/cleanup outcome.
type BrowserProviderCleanupResult struct {
	Stopped  bool
	Evidence string
	Redacted bool
}

// BrowserbaseProviderBridge is a fakeable Go port of Hermes'
// BrowserbaseProvider lifecycle. It never starts a browser; it only creates
// and releases cloud sessions through the injected HTTP client.
type BrowserbaseProviderBridge struct {
	cfg    BrowserbaseProviderConfig
	client BrowserProviderHTTPClient
}

// FirecrawlProviderBridge is a fakeable Go port of Hermes' FirecrawlProvider.
type FirecrawlProviderBridge struct {
	cfg    FirecrawlProviderConfig
	client BrowserProviderHTTPClient
}

func NewBrowserbaseProviderBridge(cfg BrowserbaseProviderConfig, client BrowserProviderHTTPClient) *BrowserbaseProviderBridge {
	if client == nil {
		client = http.DefaultClient
	}
	return &BrowserbaseProviderBridge{cfg: cfg, client: client}
}

func NewFirecrawlProviderBridge(cfg FirecrawlProviderConfig, client BrowserProviderHTTPClient) *FirecrawlProviderBridge {
	if client == nil {
		client = http.DefaultClient
	}
	return &FirecrawlProviderBridge{cfg: cfg, client: client}
}

func BrowserbaseProviderConfigFromEnv(lookup func(string) string) BrowserbaseProviderConfig {
	if lookup == nil {
		lookup = os.Getenv
	}
	cfg := BrowserbaseProviderConfig{
		APIKey:             strings.TrimSpace(lookup("BROWSERBASE_API_KEY")),
		ProjectID:          strings.TrimSpace(lookup("BROWSERBASE_PROJECT_ID")),
		BaseURL:            strings.TrimSpace(lookup("BROWSERBASE_BASE_URL")),
		EnableProxies:      CoerceBrowserSSRFGuardBool(lookup("BROWSERBASE_PROXIES"), true).Value,
		EnableKeepAlive:    CoerceBrowserSSRFGuardBool(lookup("BROWSERBASE_KEEP_ALIVE"), true).Value,
		AdvancedStealth:    CoerceBrowserSSRFGuardBool(lookup("BROWSERBASE_ADVANCED_STEALTH"), false).Value,
		UseExplicitOptions: true,
	}
	if raw := strings.TrimSpace(lookup("BROWSERBASE_SESSION_TIMEOUT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			cfg.SessionTimeoutMS = parsed
		}
	}
	return cfg
}

func FirecrawlProviderConfigFromEnv(lookup func(string) string) FirecrawlProviderConfig {
	if lookup == nil {
		lookup = os.Getenv
	}
	cfg := FirecrawlProviderConfig{
		APIKey:  strings.TrimSpace(lookup("FIRECRAWL_API_KEY")),
		BaseURL: strings.TrimSpace(lookup("FIRECRAWL_API_URL")),
	}
	if raw := strings.TrimSpace(lookup("FIRECRAWL_BROWSER_TTL")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			cfg.BrowserTTL = parsed
		}
	}
	return cfg
}

func (b *BrowserbaseProviderBridge) Configured() bool {
	return b != nil && strings.TrimSpace(b.cfg.APIKey) != "" && strings.TrimSpace(b.cfg.ProjectID) != ""
}

func (f *FirecrawlProviderBridge) Configured() bool {
	return f != nil && strings.TrimSpace(f.cfg.APIKey) != ""
}

func (b *BrowserbaseProviderBridge) CreateSession(ctx context.Context, req BrowserProviderSessionRequest) (BrowserProviderSession, error) {
	if !b.Configured() {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceUnconfigured, "Browserbase requires BROWSERBASE_API_KEY and BROWSERBASE_PROJECT_ID")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(b.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBrowserbaseBaseURL
	}

	body, features, err := b.createPayload()
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, b.redact("", err.Error()))
	}

	evidence := []string{BrowserProviderEvidenceSessionCreated}
	respBody, status, err := b.postJSON(ctx, baseURL+"/v1/sessions", body)
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, b.redact("", err.Error()))
	}
	if status == http.StatusPaymentRequired && features[BrowserProviderFeatureKeepAlive] {
		deleteJSONKey(body, "keepAlive")
		features[BrowserProviderFeatureKeepAlive] = false
		evidence = append(evidence, BrowserProviderEvidenceBrowserbasePlanFallback)
		respBody, status, err = b.postJSON(ctx, baseURL+"/v1/sessions", body)
		if err != nil {
			return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, b.redact("", err.Error()))
		}
	}
	if status == http.StatusPaymentRequired && features[BrowserProviderFeatureProxies] {
		deleteJSONKey(body, "proxies")
		features[BrowserProviderFeatureProxies] = false
		if !containsBrowserProviderString(evidence, BrowserProviderEvidenceBrowserbasePlanFallback) {
			evidence = append(evidence, BrowserProviderEvidenceBrowserbasePlanFallback)
		}
		respBody, status, err = b.postJSON(ctx, baseURL+"/v1/sessions", body)
		if err != nil {
			return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, b.redact("", err.Error()))
		}
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Browserbase create failed: HTTP %d %s", status, string(respBody))
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, b.redact("", msg))
	}

	var payload browserbaseCreateResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, b.redact("", "decode Browserbase create response: "+err.Error()))
	}
	cdpURL := firstNonEmpty(payload.ConnectURL, payload.CDPURL)
	if strings.TrimSpace(payload.ID) == "" || strings.TrimSpace(cdpURL) == "" {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, "Browserbase create response missing id or connectUrl/cdpUrl")
	}
	return BrowserProviderSession{
		ProviderName:           BrowserProviderBrowserbase,
		SessionName:            buildBrowserUseSessionName(req.TaskID, payload.ID),
		ProviderSessionID:      payload.ID,
		CompatibilitySessionID: payload.ID,
		CDPURL:                 cdpURL,
		Features:               features,
		Evidence:               evidence,
		Redacted:               true,
	}, nil
}

func (f *FirecrawlProviderBridge) CreateSession(ctx context.Context, req BrowserProviderSessionRequest) (BrowserProviderSession, error) {
	if !f.Configured() {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceUnconfigured, "Firecrawl requires FIRECRAWL_API_KEY")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(f.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultFirecrawlBaseURL
	}
	ttl := f.cfg.BrowserTTL
	if ttl <= 0 {
		ttl = defaultFirecrawlTTL
	}
	body, err := json.Marshal(map[string]any{"ttl": ttl})
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, f.redact("", err.Error()))
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v2/browser", bytes.NewReader(body))
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, f.redact("", err.Error()))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(f.cfg.APIKey))

	respBody, status, err := doBrowserProviderRequest(f.client, httpReq)
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, f.redact("", err.Error()))
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Firecrawl create failed: HTTP %d %s", status, string(respBody))
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, f.redact("", msg))
	}
	var payload firecrawlCreateResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, f.redact("", "decode Firecrawl create response: "+err.Error()))
	}
	if strings.TrimSpace(payload.ID) == "" || strings.TrimSpace(payload.CDPURL) == "" {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, "Firecrawl create response missing id or cdpUrl")
	}
	return BrowserProviderSession{
		ProviderName:           BrowserProviderFirecrawl,
		SessionName:            buildBrowserUseSessionName(req.TaskID, payload.ID),
		ProviderSessionID:      payload.ID,
		CompatibilitySessionID: payload.ID,
		CDPURL:                 payload.CDPURL,
		Features:               map[string]bool{BrowserProviderFeatureFirecrawl: true},
		Evidence:               []string{BrowserProviderEvidenceSessionCreated},
		Redacted:               true,
	}, nil
}

func (b *BrowserbaseProviderBridge) CloseSession(ctx context.Context, sessionID string) (BrowserProviderCleanupResult, error) {
	return b.closeSession(ctx, sessionID, true)
}

func (b *BrowserbaseProviderBridge) EmergencyCleanup(ctx context.Context, sessionID string) (BrowserProviderCleanupResult, error) {
	return b.closeSession(ctx, sessionID, false)
}

func (b *BrowserbaseProviderBridge) closeSession(ctx context.Context, sessionID string, requireConfigured bool) (BrowserProviderCleanupResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupSkipped, Redacted: true}, nil
	}
	if !b.Configured() {
		result := BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupSkipped, Redacted: true}
		if requireConfigured {
			return result, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, "Browserbase cleanup requires credentials")
		}
		return result, nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(b.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBrowserbaseBaseURL
	}
	body, err := json.Marshal(map[string]any{
		"projectId": strings.TrimSpace(b.cfg.ProjectID),
		"status":    "REQUEST_RELEASE",
	})
	if err != nil {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, b.redact(sessionID, err.Error()))
	}
	u := baseURL + "/v1/sessions/" + url.PathEscape(sessionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, b.redact(sessionID, err.Error()))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-BB-API-Key", strings.TrimSpace(b.cfg.APIKey))

	respBody, status, err := doBrowserProviderRequest(b.client, httpReq)
	if err != nil {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, b.redact(sessionID, err.Error()))
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Browserbase cleanup failed: HTTP %d %s", status, string(respBody))
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, b.redact(sessionID, msg))
	}
	return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupOK, Redacted: true}, nil
}

func (f *FirecrawlProviderBridge) CloseSession(ctx context.Context, sessionID string) (BrowserProviderCleanupResult, error) {
	return f.closeSession(ctx, sessionID, true)
}

func (f *FirecrawlProviderBridge) EmergencyCleanup(ctx context.Context, sessionID string) (BrowserProviderCleanupResult, error) {
	return f.closeSession(ctx, sessionID, false)
}

func (f *FirecrawlProviderBridge) closeSession(ctx context.Context, sessionID string, requireConfigured bool) (BrowserProviderCleanupResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupSkipped, Redacted: true}, nil
	}
	if !f.Configured() {
		result := BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupSkipped, Redacted: true}
		if requireConfigured {
			return result, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, "Firecrawl cleanup requires credentials")
		}
		return result, nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(f.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultFirecrawlBaseURL
	}
	u := baseURL + "/v2/browser/" + url.PathEscape(sessionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, f.redact(sessionID, err.Error()))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(f.cfg.APIKey))

	respBody, status, err := doBrowserProviderRequest(f.client, httpReq)
	if err != nil {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, f.redact(sessionID, err.Error()))
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Firecrawl cleanup failed: HTTP %d %s", status, string(respBody))
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, f.redact(sessionID, msg))
	}
	return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupOK, Redacted: true}, nil
}

func (b *BrowserbaseProviderBridge) createPayload() (map[string]any, map[string]bool, error) {
	enableKeepAlive := true
	enableProxies := true
	if b.cfg.UseExplicitOptions {
		enableKeepAlive = b.cfg.EnableKeepAlive
		enableProxies = b.cfg.EnableProxies
	}
	body := map[string]any{"projectId": strings.TrimSpace(b.cfg.ProjectID)}
	features := map[string]bool{
		BrowserProviderFeatureBasicStealth:    true,
		BrowserProviderFeatureProxies:         false,
		BrowserProviderFeatureAdvancedStealth: false,
		BrowserProviderFeatureKeepAlive:       false,
		BrowserProviderFeatureCustomTimeout:   false,
	}
	if enableKeepAlive {
		body["keepAlive"] = true
		features[BrowserProviderFeatureKeepAlive] = true
	}
	if enableProxies {
		body["proxies"] = true
		features[BrowserProviderFeatureProxies] = true
	}
	if b.cfg.AdvancedStealth {
		body["browserSettings"] = map[string]any{"advancedStealth": true}
		features[BrowserProviderFeatureAdvancedStealth] = true
	}
	if b.cfg.SessionTimeoutMS > 0 {
		body["timeout"] = b.cfg.SessionTimeoutMS
		features[BrowserProviderFeatureCustomTimeout] = true
	}
	return body, features, nil
}

func (b *BrowserbaseProviderBridge) postJSON(ctx context.Context, endpoint string, payload map[string]any) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BB-API-Key", strings.TrimSpace(b.cfg.APIKey))
	return doBrowserProviderRequest(b.client, req)
}

func doBrowserProviderRequest(client BrowserProviderHTTPClient, req *http.Request) ([]byte, int, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, defaultBrowserUseHTTPBodyLimit))
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	return body, resp.StatusCode, nil
}

func deleteJSONKey(payload map[string]any, key string) {
	delete(payload, key)
}

func (b *BrowserbaseProviderBridge) redact(sessionID, text string) string {
	if b == nil {
		return redactBrowserProviderText(text, sessionID)
	}
	return redactBrowserProviderText(text, b.cfg.APIKey, b.cfg.ProjectID, sessionID)
}

func (f *FirecrawlProviderBridge) redact(sessionID, text string) string {
	if f == nil {
		return redactBrowserProviderText(text, sessionID)
	}
	return redactBrowserProviderText(text, f.cfg.APIKey, sessionID)
}

type browserbaseCreateResponse struct {
	ID         string `json:"id"`
	ConnectURL string `json:"connectUrl"`
	CDPURL     string `json:"cdpUrl"`
}

type firecrawlCreateResponse struct {
	ID     string `json:"id"`
	CDPURL string `json:"cdpUrl"`
}

type BrowserProviderError struct {
	Evidence string
	Message  string
}

func (e *BrowserProviderError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newBrowserProviderError(evidence, message string) *BrowserProviderError {
	if evidence == BrowserProviderEvidenceCreateFailed || evidence == BrowserProviderEvidenceCleanupFailed {
		if !strings.Contains(message, BrowserProviderEvidenceProviderErrorRedacted) {
			message = strings.TrimSpace(message + " [" + BrowserProviderEvidenceProviderErrorRedacted + "]")
		}
	}
	return &BrowserProviderError{Evidence: evidence, Message: truncateBrowserBridgeText(message, defaultBrowserUseRedactedErrorPreviewLength)}
}

func BrowserProviderErrorEvidence(err error) string {
	var typed *BrowserProviderError
	if errors.As(err, &typed) {
		return typed.Evidence
	}
	return ""
}

var browserProviderBearerPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)

func redactBrowserProviderText(text string, secrets ...string) string {
	text = redactBrowserBridgeSecretsOnly(text, secrets...)
	text = browserProviderBearerPattern.ReplaceAllString(text, "Bearer [redacted]")
	text = browserBridgeURLPattern.ReplaceAllString(text, "[redacted-url]")
	return truncateBrowserBridgeText(text, defaultBrowserUseRedactedErrorPreviewLength)
}

func containsBrowserProviderString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
