package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBrowserUseConfigGates(t *testing.T) {
	cfg := BrowserUseConfigFromEnv(func(string) string { return "" })
	bridge := NewBrowserUseProviderBridge(cfg, nil)
	if bridge.Configured() {
		t.Fatalf("Configured() = true, want false without direct or managed credentials")
	}
	_, err := bridge.CreateSession(context.Background(), BrowserUseSessionRequest{TaskID: "task-1"})
	if BrowserUseErrorEvidence(err) != BrowserUseEvidenceUnconfigured {
		t.Fatalf("CreateSession evidence = %q, want %q (err=%v)", BrowserUseErrorEvidence(err), BrowserUseEvidenceUnconfigured, err)
	}

	cfg = BrowserUseConfigFromEnv(func(key string) string {
		if key == "BROWSER_USE_API_KEY" {
			return "bu-secret"
		}
		return ""
	})
	bridge = NewBrowserUseProviderBridge(cfg, &recordingBrowserUseClient{})
	if !bridge.Configured() {
		t.Fatalf("Configured() = false, want true with BROWSER_USE_API_KEY")
	}

	managed := NewBrowserUseProviderBridge(BrowserUseConfig{
		Managed:       true,
		GatewayOrigin: "https://gateway.example/browser-use",
		GatewayToken:  "nous-token",
	}, &recordingBrowserUseClient{})
	if !managed.Configured() {
		t.Fatalf("Configured() = false, want true with managed gateway state")
	}

	selected := SelectBrowserCloudProvider(
		BrowserCloudProviderCandidate{Name: BrowserProviderBrowserbase, Configured: true},
		BrowserCloudProviderCandidate{Name: BrowserProviderBrowserUse, Configured: true},
		BrowserCloudProviderCandidate{Name: BrowserProviderFirecrawl, Configured: true},
	)
	if selected.Name != BrowserProviderBrowserUse {
		t.Fatalf("selected provider = %q, want %q", selected.Name, BrowserProviderBrowserUse)
	}
}

func TestBrowserUseCreateSessionMapsCDP(t *testing.T) {
	client := &recordingBrowserUseClient{
		response: `{"id":"session-123","cdpUrl":"https://cdp.browser-use.example/json/version","liveUrl":"https://live.browser-use.example/watch","features":{"stealth":true}}`,
		headers:  http.Header{"X-External-Call-Id": []string{"external-42"}},
	}
	bridge := NewBrowserUseProviderBridge(BrowserUseConfig{
		Managed:       true,
		GatewayOrigin: "https://gateway.example/api/v3",
		GatewayToken:  "nous-token",
	}, client)

	session, err := bridge.CreateSession(context.Background(), BrowserUseSessionRequest{
		TaskID:           "research task",
		ProfileName:      "work profile",
		ProfileID:        "profile-secret-id",
		ProxyCountryCode: "de",
		TimeoutMinutes:   12,
	})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("HTTP calls = %d, want 1", client.calls)
	}
	if client.last.Method != http.MethodPost || client.last.URL.String() != "https://gateway.example/api/v3/browsers" {
		t.Fatalf("request = %s %s, want POST /browsers", client.last.Method, client.last.URL)
	}
	if got := client.last.Header.Get("X-Browser-Use-API-Key"); got != "nous-token" {
		t.Fatalf("X-Browser-Use-API-Key = %q, want managed token", got)
	}
	if got := client.last.Header.Get("X-Idempotency-Key"); !strings.HasPrefix(got, "browser-use-session-create:") {
		t.Fatalf("X-Idempotency-Key = %q, want browser-use-session-create:*", got)
	}
	for _, want := range []string{`"timeout":12`, `"proxyCountryCode":"de"`, `"profileName":"work profile"`, `"profileId":"profile-secret-id"`} {
		if !strings.Contains(client.lastBody, want) {
			t.Fatalf("payload %s missing %s", client.lastBody, want)
		}
	}

	if session.Evidence != BrowserUseEvidenceSessionCreated || !session.Redacted {
		t.Fatalf("session evidence/redaction = %q/%v", session.Evidence, session.Redacted)
	}
	if session.ProviderSessionID != "session-123" || session.CompatibilitySessionID != "session-123" {
		t.Fatalf("session IDs not mapped: %#v", session)
	}
	if session.CDPURL != "https://cdp.browser-use.example/json/version" || session.LiveURL != "https://live.browser-use.example/watch" {
		t.Fatalf("session URLs not mapped: %#v", session)
	}
	if session.ExternalCallID != "external-42" || !session.Features["browser_use"] || !session.Features["stealth"] {
		t.Fatalf("features/external call not mapped: %#v", session)
	}
	if session.Inputs.TimeoutMinutes != 12 || session.Inputs.ProxyCountryCode != "de" || !session.Inputs.ProfileNameSet || !session.Inputs.ProfileIDSet {
		t.Fatalf("session inputs not recorded safely: %#v", session.Inputs)
	}
	if strings.Contains(session.SessionName, "session-123") || strings.Contains(session.SessionName, " ") {
		t.Fatalf("session name %q leaks provider ID or contains unsafe spacing", session.SessionName)
	}
}

func TestBrowserUseStopCleanup(t *testing.T) {
	client := &recordingBrowserUseClient{response: `{}`}
	bridge := NewBrowserUseProviderBridge(BrowserUseConfig{APIKey: "bu-secret"}, client)

	result, err := bridge.StopSession(context.Background(), "session-123")
	if err != nil {
		t.Fatalf("StopSession returned error: %v", err)
	}
	if !result.Stopped || result.Evidence != BrowserUseEvidenceCleanupOK || !result.Redacted {
		t.Fatalf("cleanup result = %#v", result)
	}
	if client.last.Method != http.MethodPatch || client.last.URL.String() != defaultBrowserUseBaseURL+"/browsers/session-123" {
		t.Fatalf("cleanup request = %s %s", client.last.Method, client.last.URL)
	}
	if !strings.Contains(client.lastBody, `"action":"stop"`) {
		t.Fatalf("cleanup body = %s, want action=stop", client.lastBody)
	}

	result, err = bridge.EmergencyCleanup(context.Background(), "")
	if err != nil {
		t.Fatalf("EmergencyCleanup empty session returned error: %v", err)
	}
	if result.Evidence != BrowserUseEvidenceCleanupSkipped || result.Stopped {
		t.Fatalf("empty cleanup result = %#v", result)
	}

	failing := NewBrowserUseProviderBridge(BrowserUseConfig{APIKey: "bu-secret"}, &recordingBrowserUseClient{
		status:   403,
		response: `<html>forbidden bu-secret https://cdp.browser-use.example/json/version session-123 ` + strings.Repeat("x", 1024) + `</html>`,
	})
	_, err = failing.StopSession(context.Background(), "session-123")
	if BrowserUseErrorEvidence(err) != BrowserUseEvidenceCleanupFailed {
		t.Fatalf("cleanup evidence = %q, want %q", BrowserUseErrorEvidence(err), BrowserUseEvidenceCleanupFailed)
	}
	if errText := err.Error(); strings.Contains(errText, "bu-secret") || strings.Contains(errText, "cdp.browser-use.example") || strings.Contains(errText, "session-123") || len(errText) > 700 {
		t.Fatalf("cleanup error was not safely redacted/bounded: %q", errText)
	}
}

func TestBrowserHarnessInternalRunnerBudgetsOutput(t *testing.T) {
	backend := &recordingBrowserHarnessBackend{
		result: BrowserHarnessActionResult{
			SchemaVersion: browserHarnessActionSchemaVersion,
			Evidence:      BrowserHarnessEvidenceActionAccepted,
			Kind:          BrowserActionNavigate,
			TaskID:        "Task 7 / Login",
			URL:           "https://example.com",
			Title:         "Example",
			Text:          "page ready\n" + strings.Repeat("content ", 64),
		},
	}
	bridge := BrowserHarnessBridge{Backend: backend}

	result, err := bridge.Run(context.Background(), BrowserHarnessCommandRequest{
		TaskID:     "Task 7 / Login",
		ActionJSON: []byte(`{"schema_version":"gormes.browser.action.v1","kind":"navigate","task_id":"Task 7 / Login","url":"https://example.com","new_tab":true}`),
		Env: map[string]string{
			"BROWSER_USE_API_KEY": "bu-secret",
			"OTHER":               "ok",
		},
		Budget: ToolResultBudgetConfig{
			OutputDir:       t.TempDir(),
			TextBudgetBytes: 64,
			PreviewBytes:    512,
		},
		Action: BrowserAction{Kind: BrowserActionSnapshot, TaskID: "Task 7 / Login"},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Argv) != 0 {
		t.Fatalf("Argv = %v, want no external command argv", result.Argv)
	}
	if backend.env["BU_NAME"] != "gormes_Task_7_Login" {
		t.Fatalf("BU_NAME = %q, want sanitized Gormes session key", backend.env["BU_NAME"])
	}
	if backend.env["BROWSER_USE_API_KEY"] != "bu-secret" {
		t.Fatalf("backend did not receive BROWSER_USE_API_KEY in env")
	}
	if result.Env["BROWSER_USE_API_KEY"] == "bu-secret" {
		t.Fatalf("result env leaked BROWSER_USE_API_KEY")
	}
	if result.Evidence != BrowserHarnessEvidenceCommandOK || result.Envelope.Evidence != BrowserEvidenceResultTruncated {
		t.Fatalf("result evidence = %q/%q, want command ok + truncated envelope", result.Evidence, result.Envelope.Evidence)
	}
	if !strings.Contains(result.Envelope.Text, "tool_output_artifact") || !strings.Contains(result.Envelope.Text, "schema_version") {
		t.Fatalf("envelope text missing artifact pointer/JSON preview: %q", result.Envelope.Text)
	}
}

func TestBrowserHarnessBridgeDefaultRunsInternalBackend(t *testing.T) {
	backend := &recordingBrowserHarnessBackend{
		result: BrowserHarnessActionResult{
			SchemaVersion: browserHarnessActionSchemaVersion,
			Evidence:      BrowserHarnessEvidenceActionAccepted,
			Kind:          BrowserActionNavigate,
			TaskID:        "Task 7 / Login",
			URL:           "https://example.com",
			Title:         "Example",
			Text:          "page ready",
		},
	}
	bridge := BrowserHarnessBridge{Backend: backend}

	result, err := bridge.Run(context.Background(), BrowserHarnessCommandRequest{
		TaskID:     "Task 7 / Login",
		ActionJSON: []byte(`{"schema_version":"gormes.browser.action.v1","kind":"navigate","task_id":"Task 7 / Login","url":"https://example.com","new_tab":true}`),
		Env: map[string]string{
			"BROWSER_USE_API_KEY": "bu-secret",
			"OTHER":               "ok",
		},
		Budget: ToolResultBudgetConfig{
			OutputDir:       t.TempDir(),
			TextBudgetBytes: 4096,
			PreviewBytes:    512,
		},
		Action: BrowserAction{Kind: BrowserActionNavigate, TaskID: "Task 7 / Login", URL: "https://example.com"},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Argv) != 0 {
		t.Fatalf("Argv = %v, want no external command argv for default internal backend", result.Argv)
	}
	if backend.req.Kind != BrowserActionNavigate || backend.req.URL != "https://example.com" || !backend.req.NewTab {
		t.Fatalf("backend request = %#v, want new-tab navigate action", backend.req)
	}
	if backend.env["BU_NAME"] != "gormes_Task_7_Login" {
		t.Fatalf("BU_NAME = %q, want sanitized Gormes session key", backend.env["BU_NAME"])
	}
	if backend.env["BROWSER_USE_API_KEY"] != "bu-secret" {
		t.Fatalf("backend did not receive BROWSER_USE_API_KEY in env")
	}
	if result.Env["BROWSER_USE_API_KEY"] == "bu-secret" {
		t.Fatalf("result env leaked BROWSER_USE_API_KEY")
	}
	if result.Evidence != BrowserHarnessEvidenceCommandOK || !strings.Contains(result.Envelope.Text, "Example") {
		t.Fatalf("result = %#v", result)
	}
}

func TestBrowserHarnessChromedpBackendNavigateCreatesTargetBeforePageNavigate(t *testing.T) {
	transport := &recordingBrowserHarnessCDPTransport{
		responses: map[string]json.RawMessage{
			"Target.createTarget":   json.RawMessage(`{"targetId":"target-1"}`),
			"Target.activateTarget": json.RawMessage(`{}`),
			"Page.navigate":         json.RawMessage(`{"frameId":"frame-1"}`),
			"Runtime.evaluate":      json.RawMessage(`{"result":{"value":{"url":"https://example.com","title":"Example","text":"Ready","interactive":[],"readyState":"complete"}}}`),
		},
	}
	backend := NewBrowserHarnessChromedpBackendFromTransport(transport)

	result, err := backend.RunAction(context.Background(), BrowserHarnessActionRequest{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Kind:          BrowserActionNavigate,
		TaskID:        "nav-task",
		URL:           "https://example.com",
		NewTab:        true,
	}, nil)
	if err != nil {
		t.Fatalf("RunAction returned error: %v", err)
	}
	if got, want := strings.Join(transport.methods, ","), "Target.createTarget,Target.activateTarget,Page.navigate,Runtime.evaluate"; got != want {
		t.Fatalf("CDP methods = %s, want %s", got, want)
	}
	if result.Evidence != BrowserHarnessEvidenceActionAccepted || result.Title != "Example" || result.Text != "Ready" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBrowserHarnessLegacyCommandRunnerExplicit(t *testing.T) {
	runner := &recordingHarnessRunner{result: BrowserHarnessProcessResult{Stdout: []byte("legacy ok\n")}}
	bridge := BrowserHarnessBridge{Command: legacyBrowserHarnessCommand, Protocol: BrowserHarnessProtocolLegacy, Runner: runner}

	result, err := bridge.Run(context.Background(), BrowserHarnessCommandRequest{
		TaskID: "legacy task",
		Code:   `new_tab("https://example.com")`,
		Budget: ToolResultBudgetConfig{
			OutputDir:       t.TempDir(),
			TextBudgetBytes: 256,
			PreviewBytes:    128,
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got, want := strings.Join(runner.argv, "\x00"), "browser-harness\x00-c\x00"+`new_tab("https://example.com")`; got != want {
		t.Fatalf("argv = %q, want %q", got, want)
	}
	if result.Evidence != BrowserHarnessEvidenceCommandOK || !strings.Contains(result.Envelope.Text, "legacy ok") {
		t.Fatalf("result = %#v", result)
	}
}

func TestBrowserHarnessNoLiveRuntimeInUnitTests(t *testing.T) {
	harness := BrowserHarnessBridge{Backend: &recordingBrowserHarnessBackend{err: errors.New("synthetic failure")}}
	result, err := harness.Run(context.Background(), BrowserHarnessCommandRequest{
		TaskID:     "unit test",
		ActionJSON: []byte(`{"schema_version":"gormes.browser.action.v1","kind":"snapshot"}`),
		Budget: ToolResultBudgetConfig{
			OutputDir:       t.TempDir(),
			TextBudgetBytes: 128,
			PreviewBytes:    64,
		},
	})
	if err == nil {
		t.Fatalf("Run returned nil error, want synthetic fake-runner failure")
	}
	if result.Evidence != BrowserHarnessEvidenceCommandFailed {
		t.Fatalf("evidence = %q, want %q", result.Evidence, BrowserHarnessEvidenceCommandFailed)
	}

	httpClient := &recordingBrowserUseClient{err: errors.New("synthetic network disabled")}
	bridge := NewBrowserUseProviderBridge(BrowserUseConfig{APIKey: "bu-secret"}, httpClient)
	_, err = bridge.CreateSession(context.Background(), BrowserUseSessionRequest{TaskID: "unit test"})
	if BrowserUseErrorEvidence(err) != BrowserUseEvidenceSessionCreateFailed {
		t.Fatalf("CreateSession evidence = %q, want %q", BrowserUseErrorEvidence(err), BrowserUseEvidenceSessionCreateFailed)
	}
	if httpClient.calls != 1 {
		t.Fatalf("fake HTTP calls = %d, want 1; tests should use fake clients only", httpClient.calls)
	}
}

type recordingBrowserUseClient struct {
	calls    int
	last     *http.Request
	lastBody string
	status   int
	response string
	headers  http.Header
	err      error
}

func (c *recordingBrowserUseClient) Do(req *http.Request) (*http.Response, error) {
	c.calls++
	c.last = req.Clone(req.Context())
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	c.lastBody = string(body)
	if c.err != nil {
		return nil, c.err
	}
	status := c.status
	if status == 0 {
		status = http.StatusOK
	}
	if c.headers == nil {
		c.headers = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     c.headers,
		Body:       io.NopCloser(strings.NewReader(c.response)),
		Request:    req,
	}, nil
}

type recordingHarnessRunner struct {
	argv   []string
	env    map[string]string
	result BrowserHarnessProcessResult
	err    error
}

func (r *recordingHarnessRunner) Run(ctx context.Context, argv []string, env map[string]string) (BrowserHarnessProcessResult, error) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= 0 {
		return BrowserHarnessProcessResult{}, context.DeadlineExceeded
	}
	r.argv = append([]string(nil), argv...)
	r.env = map[string]string{}
	for k, v := range env {
		r.env[k] = v
	}
	return r.result, r.err
}

type recordingBrowserHarnessBackend struct {
	req    BrowserHarnessActionRequest
	env    map[string]string
	result BrowserHarnessActionResult
	err    error
}

func (b *recordingBrowserHarnessBackend) RunAction(_ context.Context, req BrowserHarnessActionRequest, env map[string]string) (BrowserHarnessActionResult, error) {
	b.req = req
	b.env = map[string]string{}
	for k, v := range env {
		b.env[k] = v
	}
	return b.result, b.err
}

type recordingBrowserHarnessCDPTransport struct {
	methods   []string
	params    []any
	responses map[string]json.RawMessage
	err       error
}

func (t *recordingBrowserHarnessCDPTransport) SendCommand(_ context.Context, method string, params any) (json.RawMessage, error) {
	t.methods = append(t.methods, method)
	t.params = append(t.params, params)
	if t.err != nil {
		return nil, t.err
	}
	if t.responses != nil {
		if raw, ok := t.responses[method]; ok {
			return raw, nil
		}
	}
	return json.RawMessage(`{}`), nil
}
