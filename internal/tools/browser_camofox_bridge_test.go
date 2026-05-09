package tools

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestCamofoxConfigMode(t *testing.T) {
	cfg := CamofoxProviderConfigFromEnv(func(key string) string {
		switch key {
		case "CAMOFOX_URL":
			return " http://localhost:9377/ "
		case "BROWSER_CDP_URL":
			return " "
		case "GORMES_HOME":
			return "/tmp/gormes-home"
		default:
			return ""
		}
	})
	if cfg.BaseURL != "http://localhost:9377" {
		t.Fatalf("BaseURL = %q, want trimmed Camofox URL", cfg.BaseURL)
	}
	bridge := NewCamofoxProviderBridge(cfg, &recordingBrowserProviderClient{})
	if !bridge.Configured() {
		t.Fatalf("Configured() = false, want true when CAMOFOX_URL is set and BROWSER_CDP_URL is blank")
	}

	suppressed := NewCamofoxProviderBridge(CamofoxProviderConfig{
		BaseURL: "http://localhost:9377",
		CDPURL:  "http://127.0.0.1:9222",
	}, &recordingBrowserProviderClient{})
	if suppressed.Configured() {
		t.Fatalf("Configured() = true, want false when BROWSER_CDP_URL is nonblank")
	}

	client := &recordingBrowserProviderClient{}
	unconfigured := NewCamofoxProviderBridge(CamofoxProviderConfig{}, client)
	_, err := unconfigured.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task-1"})
	if BrowserProviderErrorEvidence(err) != BrowserProviderEvidenceUnconfigured {
		t.Fatalf("CreateSession evidence = %q, want %q (err=%v)", BrowserProviderErrorEvidence(err), BrowserProviderEvidenceUnconfigured, err)
	}
	if client.calls != 0 {
		t.Fatalf("unconfigured Camofox made %d HTTP calls", client.calls)
	}
}

func TestCamofoxManagedIdentityMatchesHermesUUID5(t *testing.T) {
	identity := CamofoxManagedIdentity("/tmp/gormes-home/browser_auth/camofox", "task one")
	if identity.UserID != "hermes_2210ce4e11" {
		t.Fatalf("UserID = %q, want Hermes uuid5-derived value", identity.UserID)
	}
	if identity.SessionKey != "task_e13ee756615055e2" {
		t.Fatalf("SessionKey = %q, want Hermes uuid5-derived task key", identity.SessionKey)
	}

	sameProfileDifferentTask := CamofoxManagedIdentity("/tmp/gormes-home/browser_auth/camofox", "another")
	if sameProfileDifferentTask.UserID != identity.UserID {
		t.Fatalf("same profile UserID = %q, want %q", sameProfileDifferentTask.UserID, identity.UserID)
	}
	if sameProfileDifferentTask.SessionKey == identity.SessionKey {
		t.Fatalf("different task reused SessionKey %q", identity.SessionKey)
	}

	defaultTask := CamofoxManagedIdentity("/tmp/gormes-home/browser_auth/camofox", "")
	if defaultTask.SessionKey != "task_f4e8252e63205966" {
		t.Fatalf("default SessionKey = %q, want Hermes default task key", defaultTask.SessionKey)
	}
}

func TestCamofoxCreateSessionPostsTabWithIdentity(t *testing.T) {
	client := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{
			{status: http.StatusOK, body: `{"tabId":"tab-123","url":"about:blank"}`},
		},
	}
	bridge := NewCamofoxProviderBridge(CamofoxProviderConfig{
		BaseURL:            "http://localhost:9377",
		ManagedPersistence: true,
		StateRoot:          "/tmp/gormes-home/browser_auth/camofox",
	}, client)

	session, err := bridge.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task one"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("HTTP calls = %d, want 1", client.calls)
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.String() != "http://localhost:9377/tabs" {
		t.Fatalf("request = %s %s, want POST /tabs", req.Method, req.URL)
	}
	for _, want := range []string{`"userId":"hermes_2210ce4e11"`, `"sessionKey":"task_e13ee756615055e2"`, `"url":"about:blank"`} {
		if !strings.Contains(client.bodies[0], want) {
			t.Fatalf("body %s missing %s", client.bodies[0], want)
		}
	}
	if session.ProviderName != BrowserProviderCamofox {
		t.Fatalf("ProviderName = %q, want %q", session.ProviderName, BrowserProviderCamofox)
	}
	if session.ProviderSessionID != "tab-123" || session.CompatibilitySessionID != "task_e13ee756615055e2" {
		t.Fatalf("session IDs not mapped: %#v", session)
	}
	if session.CDPURL != "" {
		t.Fatalf("CDPURL = %q, want empty because Camofox is REST-only", session.CDPURL)
	}
	if !session.Features[BrowserProviderFeatureCamofox] || !session.Features[BrowserProviderFeatureManagedPersistence] {
		t.Fatalf("features missing camofox/managed flags: %#v", session.Features)
	}
}

func TestCamofoxSoftCleanupAndClose(t *testing.T) {
	client := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{
			{status: http.StatusOK, body: `{"tabId":"tab-1","url":"about:blank"}`},
			{status: http.StatusOK, body: `{"tabId":"tab-2","url":"about:blank"}`},
			{status: http.StatusNoContent},
		},
	}
	bridge := NewCamofoxProviderBridge(CamofoxProviderConfig{
		BaseURL:            "http://localhost:9377",
		ManagedPersistence: true,
		StateRoot:          "/tmp/gormes-home/browser_auth/camofox",
	}, client)
	if _, err := bridge.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task one"}); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	soft := bridge.SoftCleanup("task one")
	if soft.Stopped || soft.Evidence != BrowserProviderEvidenceCamofoxSoftCleanup {
		t.Fatalf("soft cleanup = %#v, want local-only cleanup evidence", soft)
	}
	if client.calls != 1 {
		t.Fatalf("soft cleanup made HTTP calls; calls=%d", client.calls)
	}

	session, err := bridge.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task one"})
	if err != nil {
		t.Fatalf("CreateSession after soft cleanup returned error: %v", err)
	}
	if session.CompatibilitySessionID != "task_e13ee756615055e2" {
		t.Fatalf("session key after soft cleanup = %q", session.CompatibilitySessionID)
	}

	closed, err := bridge.CloseTaskSession(context.Background(), "task one")
	if err != nil {
		t.Fatalf("CloseTaskSession returned error: %v", err)
	}
	if !closed.Stopped || closed.Evidence != BrowserProviderEvidenceCleanupOK || !closed.Redacted {
		t.Fatalf("close result = %#v", closed)
	}
	if req := client.requests[2]; req.Method != http.MethodDelete || req.URL.String() != "http://localhost:9377/sessions/hermes_2210ce4e11" {
		t.Fatalf("close request = %s %s, want DELETE /sessions/user", req.Method, req.URL)
	}

	unmanagedClient := &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{{status: http.StatusOK, body: `{"tabId":"tab-3"}`}},
	}
	unmanaged := NewCamofoxProviderBridge(CamofoxProviderConfig{BaseURL: "http://localhost:9377"}, unmanagedClient)
	if _, err := unmanaged.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task one"}); err != nil {
		t.Fatalf("unmanaged CreateSession returned error: %v", err)
	}
	skipped := unmanaged.SoftCleanup("task one")
	if skipped.Stopped || skipped.Evidence != BrowserProviderEvidenceCleanupSkipped {
		t.Fatalf("unmanaged soft cleanup = %#v, want skipped", skipped)
	}
	if unmanagedClient.calls != 1 {
		t.Fatalf("unmanaged soft cleanup made HTTP calls; calls=%d", unmanagedClient.calls)
	}
}

func TestCamofoxCreateSessionErrorsRedacted(t *testing.T) {
	longBody := `secret http://localhost:9377 /tmp/gormes-home/browser_auth/camofox ` + strings.Repeat("x", 2048)
	bridge := NewCamofoxProviderBridge(CamofoxProviderConfig{
		BaseURL:            "http://localhost:9377",
		ManagedPersistence: true,
		StateRoot:          "/tmp/gormes-home/browser_auth/camofox",
	}, &recordingBrowserProviderClient{
		responses: []browserProviderHTTPResponse{{status: http.StatusForbidden, body: longBody}},
	})

	_, err := bridge.CreateSession(context.Background(), BrowserProviderSessionRequest{TaskID: "task one"})
	if BrowserProviderErrorEvidence(err) != BrowserProviderEvidenceCreateFailed {
		t.Fatalf("CreateSession evidence = %q, want %q (err=%v)", BrowserProviderErrorEvidence(err), BrowserProviderEvidenceCreateFailed, err)
	}
	errText := err.Error()
	if !strings.Contains(errText, "Camofox tab create failed") {
		t.Fatalf("error text lost failure context: %q", errText)
	}
	for _, forbidden := range []string{"http://localhost:9377", "/tmp/gormes-home/browser_auth/camofox"} {
		if strings.Contains(errText, forbidden) {
			t.Fatalf("provider error leaked %q in %q", forbidden, errText)
		}
	}
	if len(errText) > 700 || !strings.Contains(errText, "[truncated]") {
		t.Fatalf("provider error was not bounded: len=%d text=%q", len(errText), errText)
	}
}
