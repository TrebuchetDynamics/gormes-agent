package tools

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBrowserProviderBridgeConfigGates(t *testing.T) {
	browserbaseClient := &recordingBrowserProviderClient{}
	browserbase := NewBrowserbaseProviderBridge(BrowserbaseProviderConfig{
		APIKey:  "bb-secret",
		BaseURL: "https://browserbase.example",
	}, browserbaseClient)
	if browserbase.Configured() {
		t.Fatalf("Browserbase Configured() = true, want false without project id")
	}
	_, err := browserbase.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task-1"})
	if BrowserProviderErrorEvidence(err) != BrowserProviderEvidenceUnconfigured {
		t.Fatalf("Browserbase CreateSession evidence = %q, want %q (err=%v)", BrowserProviderErrorEvidence(err), BrowserProviderEvidenceUnconfigured, err)
	}
	if browserbaseClient.calls != 0 {
		t.Fatalf("Browserbase made %d HTTP calls without complete config", browserbaseClient.calls)
	}

	firecrawlClient := &recordingBrowserProviderClient{}
	firecrawl := NewFirecrawlProviderBridge(FirecrawlProviderConfig{
		BaseURL: "https://firecrawl.example",
	}, firecrawlClient)
	if firecrawl.Configured() {
		t.Fatalf("Firecrawl Configured() = true, want false without api key")
	}
	_, err = firecrawl.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task-1"})
	if BrowserProviderErrorEvidence(err) != BrowserProviderEvidenceUnconfigured {
		t.Fatalf("Firecrawl CreateSession evidence = %q, want %q (err=%v)", BrowserProviderErrorEvidence(err), BrowserProviderEvidenceUnconfigured, err)
	}
	if firecrawlClient.calls != 0 {
		t.Fatalf("Firecrawl made %d HTTP calls without complete config", firecrawlClient.calls)
	}

	if !NewBrowserbaseProviderBridge(BrowserbaseProviderConfig{APIKey: "bb-secret", ProjectID: "project-123"}, nil).Configured() {
		t.Fatalf("Browserbase Configured() = false, want true with api key + project id")
	}
	if !NewFirecrawlProviderBridge(FirecrawlProviderConfig{APIKey: "fc-secret"}, nil).Configured() {
		t.Fatalf("Firecrawl Configured() = false, want true with api key")
	}
}

func TestBrowserbaseCreateSessionFallbacks(t *testing.T) {
	client := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{
			{status: http.StatusPaymentRequired, body: `{"error":"keepAlive requires paid plan"}`},
			{status: http.StatusPaymentRequired, body: `{"error":"proxies require paid plan"}`},
			{status: http.StatusOK, body: `{"id":"bb-session-123","connectUrl":"wss://browserbase.example/session/bb-session-123"}`},
		},
	}
	bridge := NewBrowserbaseProviderBridge(BrowserbaseProviderConfig{
		APIKey:             "bb-secret",
		ProjectID:          "project-123",
		BaseURL:            "https://browserbase.example",
		EnableKeepAlive:    true,
		EnableProxies:      true,
		AdvancedStealth:    true,
		SessionTimeoutMS:   90000,
		UseExplicitOptions: true,
	}, client)

	session, err := bridge.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "login task"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if client.calls != 3 {
		t.Fatalf("HTTP calls = %d, want 3 fallback attempts", client.calls)
	}
	if !strings.Contains(client.bodies[0], `"keepAlive":true`) || !strings.Contains(client.bodies[0], `"proxies":true`) {
		t.Fatalf("first payload missing keepAlive/proxies: %s", client.bodies[0])
	}
	if strings.Contains(client.bodies[1], "keepAlive") || !strings.Contains(client.bodies[1], `"proxies":true`) {
		t.Fatalf("second payload should drop keepAlive only: %s", client.bodies[1])
	}
	if strings.Contains(client.bodies[2], "keepAlive") || strings.Contains(client.bodies[2], "proxies") {
		t.Fatalf("third payload should drop keepAlive and proxies: %s", client.bodies[2])
	}
	if got := client.requests[0].Header.Get("X-BB-API-Key"); got != "bb-secret" {
		t.Fatalf("X-BB-API-Key = %q, want direct key", got)
	}
	if session.ProviderName != BrowserProviderBrowserbase || session.ProviderSessionID != "bb-session-123" || session.CompatibilitySessionID != "bb-session-123" {
		t.Fatalf("session IDs/provider not mapped: %#v", session)
	}
	if session.CDPURL != "wss://browserbase.example/session/bb-session-123" {
		t.Fatalf("CDPURL = %q", session.CDPURL)
	}
	if !session.Features["basic_stealth"] || session.Features["keep_alive"] || session.Features["proxies"] || !session.Features["advanced_stealth"] || !session.Features["custom_timeout"] {
		t.Fatalf("features did not reflect fallback state: %#v", session.Features)
	}
	if !containsBrowserProviderEvidence(session.Evidence, BrowserProviderEvidenceBrowserbasePlanFallback) {
		t.Fatalf("session evidence = %#v, want browserbase fallback evidence", session.Evidence)
	}
	if strings.Contains(session.SessionName, "bb-session-123") || strings.Contains(session.SessionName, " ") {
		t.Fatalf("session name %q leaks provider id or contains unsafe spacing", session.SessionName)
	}
}

func TestFirecrawlCreateSessionTTLAndURL(t *testing.T) {
	client := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{
			{status: http.StatusOK, body: `{"id":"fc-session-123","cdpUrl":"wss://firecrawl.example/session/fc-session-123"}`},
		},
	}
	bridge := NewFirecrawlProviderBridge(FirecrawlProviderConfig{
		APIKey:     "fc-secret",
		BaseURL:    "https://firecrawl.example/custom",
		BrowserTTL: 123,
	}, client)

	session, err := bridge.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "research"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("HTTP calls = %d, want 1", client.calls)
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.String() != "https://firecrawl.example/custom/v2/browser" {
		t.Fatalf("request = %s %s, want POST configured /v2/browser", req.Method, req.URL)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer fc-secret" {
		t.Fatalf("Authorization = %q, want bearer token", got)
	}
	if !strings.Contains(client.bodies[0], `"ttl":123`) {
		t.Fatalf("payload = %s, want ttl", client.bodies[0])
	}
	if session.ProviderName != BrowserProviderFirecrawl || session.ProviderSessionID != "fc-session-123" || session.CompatibilitySessionID != "fc-session-123" {
		t.Fatalf("session IDs/provider not mapped: %#v", session)
	}
	if session.CDPURL != "wss://firecrawl.example/session/fc-session-123" || !session.Features["firecrawl"] {
		t.Fatalf("session cdp/features not mapped: %#v", session)
	}
}

func TestBrowserProviderCloseAndEmergencyCleanup(t *testing.T) {
	browserbaseClient := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{
			{status: http.StatusNoContent},
			{status: http.StatusNoContent},
		},
	}
	browserbase := NewBrowserbaseProviderBridge(BrowserbaseProviderConfig{
		APIKey:    "bb-secret",
		ProjectID: "project-123",
		BaseURL:   "https://browserbase.example",
	}, browserbaseClient)

	result, err := browserbase.CloseSession(context.Background(), "bb-session-123")
	if err != nil {
		t.Fatalf("Browserbase CloseSession returned error: %v", err)
	}
	if !result.Stopped || result.Evidence != BrowserProviderEvidenceCleanupOK || !result.Redacted {
		t.Fatalf("Browserbase cleanup result = %#v", result)
	}
	if req := browserbaseClient.requests[0]; req.Method != http.MethodPost || req.URL.String() != "https://browserbase.example/v1/sessions/bb-session-123" {
		t.Fatalf("Browserbase cleanup request = %s %s", req.Method, req.URL)
	}
	if !strings.Contains(browserbaseClient.bodies[0], `"status":"REQUEST_RELEASE"`) || !strings.Contains(browserbaseClient.bodies[0], `"projectId":"project-123"`) {
		t.Fatalf("Browserbase cleanup body = %s", browserbaseClient.bodies[0])
	}

	result, err = browserbase.EmergencyCleanup(context.Background(), "")
	if err != nil {
		t.Fatalf("Browserbase empty EmergencyCleanup returned error: %v", err)
	}
	if result.Stopped || result.Evidence != BrowserProviderEvidenceCleanupSkipped {
		t.Fatalf("Browserbase empty cleanup result = %#v", result)
	}

	firecrawlClient := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{{status: http.StatusNoContent}},
	}
	firecrawl := NewFirecrawlProviderBridge(FirecrawlProviderConfig{
		APIKey:  "fc-secret",
		BaseURL: "https://firecrawl.example",
	}, firecrawlClient)
	result, err = firecrawl.CloseSession(context.Background(), "fc-session-123")
	if err != nil {
		t.Fatalf("Firecrawl CloseSession returned error: %v", err)
	}
	if !result.Stopped || result.Evidence != BrowserProviderEvidenceCleanupOK || !result.Redacted {
		t.Fatalf("Firecrawl cleanup result = %#v", result)
	}
	if req := firecrawlClient.requests[0]; req.Method != http.MethodDelete || req.URL.String() != "https://firecrawl.example/v2/browser/fc-session-123" {
		t.Fatalf("Firecrawl cleanup request = %s %s", req.Method, req.URL)
	}

	missingCreds := NewFirecrawlProviderBridge(FirecrawlProviderConfig{}, &recordingBrowserProviderClient{})
	result, err = missingCreds.EmergencyCleanup(context.Background(), "fc-session-123")
	if err != nil {
		t.Fatalf("Firecrawl missing-credential EmergencyCleanup returned error: %v", err)
	}
	if result.Stopped || result.Evidence != BrowserProviderEvidenceCleanupSkipped {
		t.Fatalf("Firecrawl missing-credential cleanup result = %#v", result)
	}
}

func TestBrowserProviderErrorsRedacted(t *testing.T) {
	longBody := `<html>secret bb-secret project-123 Bearer fc-secret ` +
		`wss://browserbase.example/session/bb-session-123 ` +
		strings.Repeat("x", 2048) + `</html>`
	browserbase := NewBrowserbaseProviderBridge(BrowserbaseProviderConfig{
		APIKey:    "bb-secret",
		ProjectID: "project-123",
		BaseURL:   "https://browserbase.example",
	}, &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{{status: http.StatusForbidden, body: longBody}},
	})

	_, err := browserbase.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "secret task"})
	if BrowserProviderErrorEvidence(err) != BrowserProviderEvidenceCreateFailed {
		t.Fatalf("CreateSession evidence = %q, want %q (err=%v)", BrowserProviderErrorEvidence(err), BrowserProviderEvidenceCreateFailed, err)
	}
	errText := err.Error()
	for _, forbidden := range []string{"bb-secret", "project-123", "fc-secret", "Bearer fc-secret", "browserbase.example", "bb-session-123"} {
		if strings.Contains(errText, forbidden) {
			t.Fatalf("provider error leaked %q in %q", forbidden, errText)
		}
	}
	if len(errText) > 700 || !strings.Contains(errText, "[truncated]") {
		t.Fatalf("provider error was not bounded: len=%d text=%q", len(errText), errText)
	}
}

type browserProviderHTTPResponse struct {
	status int
	body   string
}

type recordingBrowserProviderClient struct {
	calls     int
	requests  []*http.Request
	bodies    []string
	responses []browserProviderHTTPResponse
	err       error
}

func (c *recordingBrowserProviderClient) Do(req *http.Request) (*http.Response, error) {
	c.calls++
	c.requests = append(c.requests, req.Clone(req.Context()))
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}
	c.bodies = append(c.bodies, string(body))
	if c.err != nil {
		return nil, c.err
	}
	idx := c.calls - 1
	resp := browserProviderHTTPResponse{status: http.StatusOK, body: `{}`}
	if idx < len(c.responses) {
		resp = c.responses[idx]
	}
	if resp.status == 0 {
		resp.status = http.StatusOK
	}
	return &http.Response{
		StatusCode: resp.status,
		Status:     http.StatusText(resp.status),
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Request:    req,
	}, nil
}

func containsBrowserProviderEvidence(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
