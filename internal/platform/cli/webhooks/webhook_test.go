package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeWebhookURL_TrimsAndCanonicalizes(t *testing.T) {
	got, err := NormalizeWebhookURL("  HTTPS://Example.COM/Hooks/x/  ")
	if err != nil {
		t.Fatalf("NormalizeWebhookURL returned error: %v", err)
	}
	const want = "https://example.com/Hooks/x"
	if got != want {
		t.Fatalf("NormalizeWebhookURL = %q, want %q", got, want)
	}
}

func TestNormalizeWebhookURL_StripsTrailingSlashOnRoot(t *testing.T) {
	got, err := NormalizeWebhookURL("https://example.com/")
	if err != nil {
		t.Fatalf("NormalizeWebhookURL returned error: %v", err)
	}
	const want = "https://example.com"
	if got != want {
		t.Fatalf("NormalizeWebhookURL = %q, want %q", got, want)
	}
}

func TestNormalizeWebhookURL_PreservesQueryAndFragment(t *testing.T) {
	got, err := NormalizeWebhookURL("https://example.com/x/?a=1#f")
	if err != nil {
		t.Fatalf("NormalizeWebhookURL returned error: %v", err)
	}
	const want = "https://example.com/x?a=1#f"
	if got != want {
		t.Fatalf("NormalizeWebhookURL = %q, want %q", got, want)
	}
}

func TestNormalizeWebhookURL_RejectsEmpty(t *testing.T) {
	got, err := NormalizeWebhookURL("   ")
	if got != "" {
		t.Fatalf("NormalizeWebhookURL = %q, want empty string", got)
	}
	if !errors.Is(err, ErrWebhookURLEmpty) {
		t.Fatalf("NormalizeWebhookURL err = %v, want ErrWebhookURLEmpty", err)
	}
}

func TestNormalizeWebhookURL_RejectsNonHTTP(t *testing.T) {
	cases := []string{
		"ftp://example.com",
		"javascript:alert(1)",
	}
	for _, raw := range cases {
		got, err := NormalizeWebhookURL(raw)
		if got != "" {
			t.Errorf("NormalizeWebhookURL(%q) = %q, want empty string", raw, got)
		}
		if !errors.Is(err, ErrWebhookURLBadScheme) {
			t.Errorf("NormalizeWebhookURL(%q) err = %v, want ErrWebhookURLBadScheme", raw, err)
		}
	}
}

func TestNormalizeWebhookURL_RejectsUserInfo(t *testing.T) {
	got, err := NormalizeWebhookURL("https://user:pass@example.com/x")
	if got != "" {
		t.Fatalf("NormalizeWebhookURL = %q, want empty string", got)
	}
	if !errors.Is(err, ErrWebhookURLUserInfoForbidden) {
		t.Fatalf("NormalizeWebhookURL err = %v, want ErrWebhookURLUserInfoForbidden", err)
	}
}

func TestDispatchWebhookCommandShowsUsageBeforeEnabledCheck(t *testing.T) {
	t.Parallel()

	called := false
	out, err := DispatchWebhookCommand("", map[string]any{}, "/tmp/gormes-home", WebhookCommandHandlers{
		Subscribe: func() error { called = true; return nil },
	})
	if err != nil {
		t.Fatalf("DispatchWebhookCommand: %v", err)
	}
	if called {
		t.Fatalf("empty action must not call handlers")
	}
	if !strings.Contains(out, "Usage: gormes webhook {subscribe|list|remove|test}") || !strings.Contains(out, "gormes webhook --help") {
		t.Fatalf("empty action output missing usage/help:\n%s", out)
	}
	if strings.Contains(out, "Webhook platform is not enabled") || strings.Contains(out, "hermes webhook") {
		t.Fatalf("empty action output should be usage-only Gormes text:\n%s", out)
	}
}

func TestDispatchWebhookCommandDisabledReturnsSetupHintWithoutHandler(t *testing.T) {
	t.Parallel()

	called := false
	out, err := DispatchWebhookCommand("subscribe", map[string]any{}, "/tmp/gormes-home", WebhookCommandHandlers{
		Subscribe: func() error { called = true; return nil },
	})
	if err != nil {
		t.Fatalf("DispatchWebhookCommand: %v", err)
	}
	if called {
		t.Fatalf("disabled webhook must not call handlers")
	}
	if !strings.Contains(out, "Webhook platform is not enabled") || !strings.Contains(out, "gormes gateway setup") {
		t.Fatalf("disabled action output missing setup hint:\n%s", out)
	}
}

func TestDispatchWebhookCommandAliasesToCanonicalHandlers(t *testing.T) {
	t.Parallel()

	enabled := map[string]any{"platforms": map[string]any{"webhook": map[string]any{"enabled": true}}}
	cases := []struct {
		action string
		want   string
	}{
		{action: "subscribe", want: "subscribe"},
		{action: "add", want: "subscribe"},
		{action: "list", want: "list"},
		{action: "ls", want: "list"},
		{action: "remove", want: "remove"},
		{action: "rm", want: "remove"},
		{action: "test", want: "test"},
	}
	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			called := ""
			out, err := DispatchWebhookCommand(tc.action, enabled, "/tmp/gormes-home", WebhookCommandHandlers{
				Subscribe: func() error { called = "subscribe"; return nil },
				List:      func() error { called = "list"; return nil },
				Remove:    func() error { called = "remove"; return nil },
				Test:      func() error { called = "test"; return nil },
			})
			if err != nil {
				t.Fatalf("DispatchWebhookCommand(%q): %v", tc.action, err)
			}
			if out != "" {
				t.Fatalf("DispatchWebhookCommand(%q) output = %q, want empty", tc.action, out)
			}
			if called != tc.want {
				t.Fatalf("DispatchWebhookCommand(%q) called %q, want %q", tc.action, called, tc.want)
			}
		})
	}
}

func TestListWebhookSubscriptionsEmptyShowsCreateGuidance(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SubscriptionsFilename)
	out, err := ListWebhookSubscriptions(WebhookListOptions{SubscriptionsPath: path})
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions empty: %v", err)
	}
	for _, want := range []string{
		"No dynamic webhook subscriptions.",
		"Create one with: gormes webhook subscribe <name>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("empty list output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hermes webhook") {
		t.Fatalf("empty list output should use Gormes command wording:\n%s", out)
	}
}

func TestListWebhookSubscriptionsRendersDynamicRoutesWithoutSecrets(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if err := SaveSubscriptions(path, Subscriptions{
		"deploy": {
			"description": "Deploy hook",
			"events":      []any{"push", "release"},
			"secret":      "whsec-deploy",
			"deliver":     "telegram",
		},
		"direct": {
			"events":       []any{},
			"secret":       "whsec-direct",
			"deliver":      "discord",
			"deliver_only": true,
		},
	}); err != nil {
		t.Fatalf("seed subscriptions: %v", err)
	}
	cfg := map[string]any{
		"platforms": map[string]any{
			"webhook": map[string]any{
				"extra": map[string]any{"host": "0.0.0.0", "port": 9001},
			},
		},
	}

	out, err := ListWebhookSubscriptions(WebhookListOptions{SubscriptionsPath: path, Config: cfg})
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions: %v", err)
	}
	for _, want := range []string{
		"2 webhook subscription(s)",
		"◆ deploy",
		"Deploy hook",
		"URL:     http://localhost:9001/webhooks/deploy",
		"Events:  push, release",
		"Deliver: telegram",
		"◆ direct",
		"URL:     http://localhost:9001/webhooks/direct",
		"Events:  (all)",
		"Deliver: discord (direct — no agent)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "whsec-") {
		t.Fatalf("list output leaked subscription secret:\n%s", out)
	}
}

func TestSendWebhookTestPostsSignedDefaultPayload(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if err := SaveSubscriptions(path, Subscriptions{
		"deploy": {"secret": "whsec-deploy"},
	}); err != nil {
		t.Fatalf("seed subscriptions: %v", err)
	}

	var gotPath, gotBody, gotSignature, gotEvent, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSignature = r.Header.Get("X-Hub-Signature-256")
		gotEvent = r.Header.Get("X-GitHub-Event")
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		gotBody = string(body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := webhookConfigForServer(t, server.URL)
	out, err := SendWebhookTest(WebhookTestOptions{
		SubscriptionsPath: path,
		Config:            cfg,
		Name:              " Deploy ",
	})
	if err != nil {
		t.Fatalf("SendWebhookTest: %v", err)
	}

	const defaultPayload = `{"test": true, "event_type": "test", "message": "Hello from gormes webhook test"}`
	if gotPath != "/webhooks/deploy" {
		t.Fatalf("test request path = %q, want /webhooks/deploy", gotPath)
	}
	if gotBody != defaultPayload {
		t.Fatalf("test request body = %q, want %q", gotBody, defaultPayload)
	}
	if gotEvent != "test" {
		t.Fatalf("test event header = %q, want test", gotEvent)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", gotContentType)
	}
	if wantSig := webhookTestSignature("whsec-deploy", defaultPayload); gotSignature != wantSig {
		t.Fatalf("signature = %q, want %q", gotSignature, wantSig)
	}
	for _, want := range []string{
		"Sending test POST to " + WebhookBaseURL(cfg) + "/webhooks/deploy",
		"Response (200): {\"ok\":true}",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hermes gateway") {
		t.Fatalf("test output should use Gormes wording:\n%s", out)
	}
}

func TestSendWebhookTestMissingSubscriptionDoesNotPost(t *testing.T) {
	t.Parallel()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	out, err := SendWebhookTest(WebhookTestOptions{
		SubscriptionsPath: filepath.Join(t.TempDir(), SubscriptionsFilename),
		Config:            webhookConfigForServer(t, server.URL),
		Name:              "Missing",
	})
	if err != nil {
		t.Fatalf("SendWebhookTest missing: %v", err)
	}
	if called {
		t.Fatalf("missing subscription should not POST")
	}
	if !strings.Contains(out, "No subscription named 'missing'.") {
		t.Fatalf("missing output lacks subscription guidance:\n%s", out)
	}
}

func TestSendWebhookTestHTTPErrorShowsGatewayRunHint(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if err := SaveSubscriptions(path, Subscriptions{"deploy": {"secret": "whsec-deploy"}}); err != nil {
		t.Fatalf("seed subscriptions: %v", err)
	}

	out, err := SendWebhookTest(WebhookTestOptions{
		SubscriptionsPath: path,
		Config:            map[string]any{"platforms": map[string]any{"webhook": map[string]any{"extra": map[string]any{"host": "127.0.0.1", "port": "1"}}}},
		Name:              "deploy",
	})
	if err != nil {
		t.Fatalf("SendWebhookTest http error: %v", err)
	}
	for _, want := range []string{
		"Sending test POST to http://127.0.0.1:1/webhooks/deploy",
		"Error:",
		"Is the gateway running? (gormes gateway run)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("http error output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hermes gateway") {
		t.Fatalf("http error output should use Gormes wording:\n%s", out)
	}
}

func TestRemoveWebhookSubscriptionDeletesExistingRoute(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if err := SaveSubscriptions(path, Subscriptions{
		"deploy": {"secret": "whsec-deploy"},
		"keep":   {"secret": "whsec-keep"},
	}); err != nil {
		t.Fatalf("seed subscriptions: %v", err)
	}

	out, err := RemoveWebhookSubscription(WebhookRemoveOptions{
		SubscriptionsPath: path,
		Name:              " Deploy ",
	})
	if err != nil {
		t.Fatalf("RemoveWebhookSubscription: %v", err)
	}
	if !strings.Contains(out, "Removed webhook subscription: deploy") {
		t.Fatalf("remove output missing confirmation:\n%s", out)
	}

	loaded := LoadSubscriptions(path)
	if _, ok := loaded["deploy"]; ok {
		t.Fatalf("deploy subscription still present: %#v", loaded)
	}
	if got := loaded["keep"]["secret"]; got != "whsec-keep" {
		t.Fatalf("keep subscription secret = %#v, want whsec-keep; loaded=%#v", got, loaded)
	}
	if mode := fileModePerm(t, path); mode != SubscriptionsFileMode {
		t.Fatalf("subscription file mode after remove = %#o, want %#o", mode, SubscriptionsFileMode)
	}
}

func TestRemoveWebhookSubscriptionMissingShowsStaticRouteHint(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), SubscriptionsFilename)
	out, err := RemoveWebhookSubscription(WebhookRemoveOptions{
		SubscriptionsPath: path,
		Name:              " Missing ",
	})
	if err != nil {
		t.Fatalf("RemoveWebhookSubscription missing: %v", err)
	}
	for _, want := range []string{
		"No subscription named 'missing'",
		"Static routes from config.toml cannot be removed here",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "config.yaml") || strings.Contains(out, "hermes") {
		t.Fatalf("missing output should use Gormes config wording:\n%s", out)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing remove should not create subscriptions file, stat err=%v", err)
	}
}

func TestWebhookSetupHintUsesGormesCommandsAndConfigLocation(t *testing.T) {
	t.Parallel()

	hint := WebhookSetupHint("/tmp/gormes-home")
	for _, want := range []string{
		"Webhook platform is not enabled",
		"gormes gateway setup",
		"/tmp/gormes-home/config.toml",
		"[platforms.webhook]",
		"WEBHOOK_ENABLED=true",
		"WEBHOOK_PORT=8644",
		"WEBHOOK_SECRET=your-global-secret",
		"gormes gateway run",
	} {
		if !strings.Contains(hint, want) {
			t.Fatalf("WebhookSetupHint missing %q:\n%s", want, hint)
		}
	}
	if strings.Contains(hint, "hermes gateway") || strings.Contains(hint, "/config.yaml") {
		t.Fatalf("WebhookSetupHint should use Gormes commands/config, got:\n%s", hint)
	}
}

func TestWebhookConfigDetectionEnabledAndBaseURL(t *testing.T) {
	t.Parallel()

	cfg := map[string]any{
		"platforms": map[string]any{
			"webhook": map[string]any{
				"enabled": true,
				"extra": map[string]any{
					"host": "0.0.0.0",
					"port": 9001,
				},
			},
		},
	}

	webhook := ExtractWebhookConfig(cfg)
	if got := webhook["enabled"]; got != true {
		t.Fatalf("ExtractWebhookConfig enabled = %#v, want true", got)
	}
	if !WebhookEnabled(cfg) {
		t.Fatalf("WebhookEnabled = false, want true")
	}
	if got := WebhookBaseURL(cfg); got != "http://localhost:9001" {
		t.Fatalf("WebhookBaseURL = %q, want http://localhost:9001", got)
	}
}

func TestWebhookConfigDefaultsWhenMissingOrMalformed(t *testing.T) {
	t.Parallel()

	cases := []map[string]any{
		nil,
		{},
		{"platforms": "not a map"},
		{"platforms": map[string]any{"webhook": "not a map"}},
	}
	for _, cfg := range cases {
		if got := ExtractWebhookConfig(cfg); len(got) != 0 {
			t.Fatalf("ExtractWebhookConfig(%#v) = %#v, want empty", cfg, got)
		}
		if WebhookEnabled(cfg) {
			t.Fatalf("WebhookEnabled(%#v) = true, want false", cfg)
		}
		if got := WebhookBaseURL(cfg); got != "http://localhost:8644" {
			t.Fatalf("WebhookBaseURL(%#v) = %q, want default", cfg, got)
		}
	}
}

func TestWebhookBaseURLUsesConfiguredHostAndStringPort(t *testing.T) {
	t.Parallel()

	cfg := map[string]any{
		"platforms": map[string]any{
			"webhook": map[string]any{
				"extra": map[string]any{
					"host": "webhooks.example.test",
					"port": "9443",
				},
			},
		},
	}
	if got := WebhookBaseURL(cfg); got != "http://webhooks.example.test:9443" {
		t.Fatalf("WebhookBaseURL = %q, want http://webhooks.example.test:9443", got)
	}
}

func TestLoadSubscriptionsReturnsEmptyForMissingMalformedOrNonObject(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if got := LoadSubscriptions(missing); len(got) != 0 {
		t.Fatalf("missing subscriptions = %#v, want empty", got)
	}

	malformed := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if err := os.WriteFile(malformed, []byte(`{"broken"`), 0o600); err != nil {
		t.Fatalf("write malformed fixture: %v", err)
	}
	if got := LoadSubscriptions(malformed); len(got) != 0 {
		t.Fatalf("malformed subscriptions = %#v, want empty", got)
	}

	nonObject := filepath.Join(t.TempDir(), SubscriptionsFilename)
	if err := os.WriteFile(nonObject, []byte(`["not", "an", "object"]`), 0o600); err != nil {
		t.Fatalf("write non-object fixture: %v", err)
	}
	if got := LoadSubscriptions(nonObject); len(got) != 0 {
		t.Fatalf("non-object subscriptions = %#v, want empty", got)
	}
}

func TestSaveSubscriptionsRoundTripsAndRestrictsMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", SubscriptionsFilename)
	subs := Subscriptions{
		"deploy": {
			"description": "deploy hook",
			"events":      []any{"push", "release"},
			"secret":      "whsec-test",
			"deliver":     "telegram",
		},
	}
	if err := SaveSubscriptions(path, subs); err != nil {
		t.Fatalf("SaveSubscriptions: %v", err)
	}

	mode := fileModePerm(t, path)
	if mode != SubscriptionsFileMode {
		t.Fatalf("saved mode = %#o, want %#o", mode, SubscriptionsFileMode)
	}

	loaded := LoadSubscriptions(path)
	if got := loaded["deploy"]["secret"]; got != "whsec-test" {
		t.Fatalf("loaded secret = %#v, want whsec-test; loaded=%#v", got, loaded)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod existing broad mode: %v", err)
	}
	if err := SaveSubscriptions(path, loaded); err != nil {
		t.Fatalf("SaveSubscriptions existing file: %v", err)
	}
	if mode := fileModePerm(t, path); mode != SubscriptionsFileMode {
		t.Fatalf("rewritten mode = %#o, want %#o", mode, SubscriptionsFileMode)
	}
}

func fileModePerm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

func webhookConfigForServer(t *testing.T, serverURL string) map[string]any {
	t.Helper()
	host, port, err := net.SplitHostPort(strings.TrimPrefix(serverURL, "http://"))
	if err != nil {
		t.Fatalf("split test server URL %q: %v", serverURL, err)
	}
	return map[string]any{"platforms": map[string]any{"webhook": map[string]any{"extra": map[string]any{"host": host, "port": port}}}}
}

func webhookTestSignature(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
