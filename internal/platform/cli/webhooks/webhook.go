package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Sentinel errors returned by NormalizeWebhookURL. Callers should branch on
// these via errors.Is so wrapping context can be added without breaking
// classification.
var (
	ErrWebhookURLEmpty             = errors.New("webhook URL is empty")
	ErrWebhookURLBadScheme         = errors.New("webhook URL must use http or https scheme")
	ErrWebhookURLUserInfoForbidden = errors.New("webhook URL must not embed userinfo credentials")
	ErrWebhookURLParseFailed       = errors.New("webhook URL is not parsable")
)

const (
	SubscriptionsFilename = "webhook_subscriptions.json"
	SubscriptionsFileMode = os.FileMode(0o600)
)

type Subscription map[string]any

type Subscriptions map[string]Subscription

// ExtractWebhookConfig returns the nested platforms.webhook configuration map.
// It mirrors hermes-agent/hermes_cli/webhook.py _get_webhook_config's
// fail-closed behavior by returning an empty map for missing or malformed
// configuration shapes.
func ExtractWebhookConfig(config map[string]any) map[string]any {
	platforms, ok := config["platforms"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	webhook, ok := platforms["webhook"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return webhook
}

// WebhookEnabled reports whether the extracted webhook config is truthy.
func WebhookEnabled(config map[string]any) bool {
	return webhookTruthy(ExtractWebhookConfig(config)["enabled"])
}

// WebhookBaseURL derives the operator-facing webhook URL from config. Hermes
// defaults host to 0.0.0.0 and port to 8644, but displays 0.0.0.0 as localhost.
func WebhookBaseURL(config map[string]any) string {
	webhook := ExtractWebhookConfig(config)
	extra, ok := webhook["extra"].(map[string]any)
	if !ok {
		extra = map[string]any{}
	}
	host := webhookString(extra["host"], "0.0.0.0")
	port := webhookString(extra["port"], "8644")
	if host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

func webhookString(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func webhookTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return true
	}
}

// WebhookCommandHandlers contains the concrete operations behind the Hermes
// webhook command actions. Keeping this as a small injected interface lets CLI
// wiring reuse the source-backed dispatch policy without coupling the pure
// helper to Cobra or filesystem state.
type WebhookCommandHandlers struct {
	Subscribe func() error
	List      func() error
	Remove    func() error
	Test      func() error
}

// DispatchWebhookCommand ports hermes-agent/hermes_cli/webhook.py
// webhook_command: show usage when no action is selected, require the webhook
// platform before running an action, and route aliases to their canonical
// handlers.
func DispatchWebhookCommand(action string, config map[string]any, displayHome string, handlers WebhookCommandHandlers) (string, error) {
	sub := strings.TrimSpace(strings.ToLower(action))
	if sub == "" {
		return WebhookCommandUsage(), nil
	}
	if !WebhookEnabled(config) {
		return WebhookSetupHint(displayHome), nil
	}

	switch sub {
	case "subscribe", "add":
		return "", callWebhookHandler(handlers.Subscribe)
	case "list", "ls":
		return "", callWebhookHandler(handlers.List)
	case "remove", "rm":
		return "", callWebhookHandler(handlers.Remove)
	case "test":
		return "", callWebhookHandler(handlers.Test)
	default:
		return "", nil
	}
}

func WebhookCommandUsage() string {
	return "Usage: gormes webhook {subscribe|list|remove|test}\nRun 'gormes webhook --help' for details.\n"
}

func callWebhookHandler(handler func() error) error {
	if handler == nil {
		return nil
	}
	return handler()
}

// WebhookListOptions contains the dynamic subscription file path and config
// used by the Hermes-compatible webhook list command.
type WebhookListOptions struct {
	SubscriptionsPath string
	Config            map[string]any
}

// ListWebhookSubscriptions ports hermes-agent/hermes_cli/webhook.py _cmd_list.
// It renders dynamic subscriptions with their route URLs and delivery metadata
// while intentionally omitting per-route HMAC secrets.
func ListWebhookSubscriptions(opts WebhookListOptions) (string, error) {
	subs := LoadSubscriptions(opts.SubscriptionsPath)
	if len(subs) == 0 {
		return "  No dynamic webhook subscriptions.\n  Create one with: gormes webhook subscribe <name>\n", nil
	}

	names := make([]string, 0, len(subs))
	for name := range subs {
		names = append(names, name)
	}
	sort.Strings(names)

	baseURL := WebhookBaseURL(opts.Config)
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %d webhook subscription(s):\n\n", len(subs))
	for _, name := range names {
		route := subs[name]
		events := webhookEventsString(route["events"])
		deliver := webhookString(route["deliver"], "log")
		if webhookTruthy(route["deliver_only"]) {
			deliver = fmt.Sprintf("%s (direct — no agent)", deliver)
		}
		desc := strings.TrimSpace(webhookString(route["description"], ""))

		fmt.Fprintf(&b, "  ◆ %s\n", name)
		if desc != "" {
			fmt.Fprintf(&b, "    %s\n", desc)
		}
		fmt.Fprintf(&b, "    URL:     %s/webhooks/%s\n", baseURL, name)
		fmt.Fprintf(&b, "    Events:  %s\n", events)
		fmt.Fprintf(&b, "    Deliver: %s\n\n", deliver)
	}
	return b.String(), nil
}

func webhookEventsString(value any) string {
	var events []string
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			events = append(events, strings.TrimSpace(fmt.Sprint(item)))
		}
	case []string:
		for _, item := range v {
			events = append(events, strings.TrimSpace(item))
		}
	case string:
		events = append(events, strings.TrimSpace(v))
	}
	filtered := events[:0]
	for _, event := range events {
		if event != "" {
			filtered = append(filtered, event)
		}
	}
	if len(filtered) == 0 {
		return "(all)"
	}
	return strings.Join(filtered, ", ")
}

const DefaultWebhookTestPayload = `{"test": true, "event_type": "test", "message": "Hello from gormes webhook test"}`

// WebhookTestOptions contains the dynamic subscription file path, webhook
// config, route name, optional payload, and optional HTTP client used by the
// Hermes-compatible webhook test command.
type WebhookTestOptions struct {
	SubscriptionsPath string
	Config            map[string]any
	Name              string
	Payload           string
	Client            *http.Client
}

// SendWebhookTest ports hermes-agent/hermes_cli/webhook.py _cmd_test. It sends
// a signed JSON POST to a dynamic webhook route and reports success or gateway
// reachability guidance without exposing the route secret.
func SendWebhookTest(opts WebhookTestOptions) (string, error) {
	name := strings.TrimSpace(strings.ToLower(opts.Name))
	subs := LoadSubscriptions(opts.SubscriptionsPath)
	route, ok := subs[name]
	if !ok {
		return fmt.Sprintf("  No subscription named '%s'.\n", name), nil
	}

	payload := opts.Payload
	if payload == "" {
		payload = DefaultWebhookTestPayload
	}
	secret := webhookString(route["secret"], "")
	url := fmt.Sprintf("%s/webhooks/%s", WebhookBaseURL(opts.Config), name)

	var out strings.Builder
	fmt.Fprintf(&out, "  Sending test POST to %s\n", url)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		writeWebhookTestError(&out, err)
		return out.String(), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", webhookSignature(secret, payload))
	req.Header.Set("X-GitHub-Event", "test")

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		writeWebhookTestError(&out, err)
		return out.String(), nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeWebhookTestError(&out, err)
		return out.String(), nil
	}
	fmt.Fprintf(&out, "  Response (%d): %s\n", resp.StatusCode, string(body))
	return out.String(), nil
}

func webhookSignature(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("sha256=%x", mac.Sum(nil))
}

func writeWebhookTestError(out *strings.Builder, err error) {
	fmt.Fprintf(out, "  Error: %v\n", err)
	out.WriteString("  Is the gateway running? (gormes gateway run)\n")
}

// WebhookRemoveOptions contains the dynamic subscription file path and route
// name used by the Hermes-compatible webhook remove command.
type WebhookRemoveOptions struct {
	SubscriptionsPath string
	Name              string
}

// RemoveWebhookSubscription ports hermes-agent/hermes_cli/webhook.py
// _cmd_remove. Dynamic subscriptions are removable from the persisted
// webhook_subscriptions.json file; static config routes are only explained to
// the operator so the command never mutates config.toml by surprise.
func RemoveWebhookSubscription(opts WebhookRemoveOptions) (string, error) {
	name := strings.TrimSpace(strings.ToLower(opts.Name))
	subs := LoadSubscriptions(opts.SubscriptionsPath)
	if _, ok := subs[name]; !ok {
		return fmt.Sprintf("  No subscription named '%s'.\n  Note: Static routes from config.toml cannot be removed here.\n", name), nil
	}

	delete(subs, name)
	if err := SaveSubscriptions(opts.SubscriptionsPath, subs); err != nil {
		return "", err
	}
	return fmt.Sprintf("  Removed webhook subscription: %s\n", name), nil
}

// WebhookSetupHint renders the setup guidance shown when the webhook platform
// is disabled. It ports Hermes' _setup_hint shape while using Gormes commands
// and the Gormes TOML config path.
func WebhookSetupHint(displayHome string) string {
	home := strings.TrimRight(strings.TrimSpace(displayHome), "/")
	if home == "" {
		home = "$GORMES_HOME"
	}
	return fmt.Sprintf(`
  Webhook platform is not enabled. To set it up:

  1. Run the gateway setup wizard:
     gormes gateway setup

  2. Or manually add to %s/config.toml:
     [platforms.webhook]
     enabled = true
     [platforms.webhook.extra]
     host = "0.0.0.0"
     port = 8644
     secret = "your-global-hmac-secret"

  3. Or set environment variables in %s/.env:
     WEBHOOK_ENABLED=true
     WEBHOOK_PORT=8644
     WEBHOOK_SECRET=your-global-secret

  Then start the gateway: gormes gateway run
`, home, home)
}

// LoadSubscriptions reads a Hermes-compatible dynamic webhook subscription
// file. It mirrors hermes-agent/hermes_cli/webhook.py _load_subscriptions:
// missing, malformed, or non-object files degrade to an empty subscription map
// instead of failing the caller.
func LoadSubscriptions(path string) Subscriptions {
	body, err := os.ReadFile(path)
	if err != nil {
		return Subscriptions{}
	}
	var subs Subscriptions
	if err := json.Unmarshal(body, &subs); err != nil || subs == nil {
		return Subscriptions{}
	}
	return subs
}

// SaveSubscriptions writes dynamic webhook subscriptions atomically. The file
// may contain per-route HMAC secrets, so the temp file and final destination are
// both forced to 0600 to avoid inheriting a permissive umask or existing mode.
func SaveSubscriptions(path string, subs Subscriptions) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(SubscriptionsFileMode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return os.Chmod(path, SubscriptionsFileMode)
}

// NormalizeWebhookURL canonicalizes an operator-supplied webhook URL without
// touching the network. It trims surrounding whitespace, requires an http or
// https scheme, lowercases the host, strips trailing slashes from the path
// (including reducing a "/" root to no path at all), preserves the query and
// fragment, and refuses URLs that embed userinfo credentials. The returned
// error is one of the sentinel values declared above so callers can branch
// on the typed failure mode instead of inspecting strings.
func NormalizeWebhookURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrWebhookURLEmpty
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrWebhookURLParseFailed, err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrWebhookURLBadScheme
	}
	u.Scheme = scheme

	if u.User != nil {
		return "", ErrWebhookURLUserInfoForbidden
	}

	u.Host = strings.ToLower(u.Host)

	for strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	for strings.HasSuffix(u.RawPath, "/") {
		u.RawPath = strings.TrimSuffix(u.RawPath, "/")
	}

	return u.String(), nil
}
