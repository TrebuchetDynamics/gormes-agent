package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Deterministic helpers ported from hermes_cli/webhook.py (Phase 5.O). The
// interactive subcommand surface (argparse wiring, stdout printing, HTTP
// self-test via urllib) stays intentionally unported — frontends wire their
// own flag parsers around these pure primitives, and the webhook gateway's
// hot-reload contract is the only cross-implementation invariant worth
// pinning here.

// ErrInvalidWebhookName is returned by NormalizeWebhookName for names that
// cannot be coerced into the canonical `^[a-z0-9][a-z0-9_-]*$` form.
var ErrInvalidWebhookName = errors.New("invalid webhook subscription name")

// DefaultWebhookPort mirrors the upstream config default (8644) used by
// _get_webhook_base_url when the webhook platform block omits an explicit
// port.
const DefaultWebhookPort = 8644

// DefaultWebhookDeliver mirrors `args.deliver or "log"` in _cmd_subscribe.
const DefaultWebhookDeliver = "log"

// webhookNamePattern is the byte-identical equivalent of the upstream
// re.match(r'^[a-z0-9][a-z0-9_-]*$', name) check.
var webhookNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// NormalizeWebhookName mirrors the normalize+validate step in
// hermes_cli/webhook.py::_cmd_subscribe: trim surrounding whitespace,
// lowercase, replace spaces with hyphens, then reject anything that does not
// match the canonical regex. Returns ErrInvalidWebhookName on failure.
func NormalizeWebhookName(raw string) (string, error) {
	n := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), " ", "-"))
	if !webhookNamePattern.MatchString(n) {
		return "", fmt.Errorf("%w: %q", ErrInvalidWebhookName, raw)
	}
	return n, nil
}

// FormatWebhookBaseURL mirrors hermes_cli/webhook.py::_get_webhook_base_url:
// a wildcard/empty host is displayed as "localhost" so the printed URL stays
// clickable, and a zero port falls back to DefaultWebhookPort.
func FormatWebhookBaseURL(host string, port int) string {
	display := host
	if display == "" || display == "0.0.0.0" {
		display = "localhost"
	}
	if port == 0 {
		port = DefaultWebhookPort
	}
	return fmt.Sprintf("http://%s:%d", display, port)
}

// WebhookSubscription is the serialized route payload persisted at
// `${GORMES_HOME}/webhook_subscriptions.json`. JSON field tags match the
// upstream key names exactly so the webhook gateway's hot-reload watcher
// consumes the same file produced by either implementation.
type WebhookSubscription struct {
	Description  string         `json:"description"`
	Events       []string       `json:"events"`
	Secret       string         `json:"secret"`
	Prompt       string         `json:"prompt"`
	Skills       []string       `json:"skills"`
	Deliver      string         `json:"deliver"`
	CreatedAt    string         `json:"created_at"`
	DeliverExtra map[string]any `json:"deliver_extra,omitempty"`
}

// WebhookSubscriptionOpts mirrors the CLI flag bundle consumed by the
// upstream `_cmd_subscribe` function. Name is expected to already be the
// canonical form returned by NormalizeWebhookName.
type WebhookSubscriptionOpts struct {
	Name          string
	Description   string
	Events        string // comma-separated list mirroring --events
	Secret        string
	Prompt        string
	Skills        string // comma-separated list mirroring --skills
	Deliver       string
	DeliverChatID string
}

// BuildWebhookSubscription ports the route-dict construction in
// hermes_cli/webhook.py::_cmd_subscribe. Empty CSV lists become empty slices
// (not nil) so the JSON output stays `[]` instead of `null`, preserving a
// byte-identical on-disk shape with the upstream writer.
func BuildWebhookSubscription(opts WebhookSubscriptionOpts, now time.Time) WebhookSubscription {
	description := opts.Description
	if description == "" {
		description = fmt.Sprintf("Agent-created subscription: %s", opts.Name)
	}
	deliver := opts.Deliver
	if deliver == "" {
		deliver = DefaultWebhookDeliver
	}
	sub := WebhookSubscription{
		Description: description,
		Events:      splitTrimCSV(opts.Events),
		Secret:      opts.Secret,
		Prompt:      opts.Prompt,
		Skills:      splitTrimCSV(opts.Skills),
		Deliver:     deliver,
		CreatedAt:   now.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if opts.DeliverChatID != "" {
		sub.DeliverExtra = map[string]any{"chat_id": opts.DeliverChatID}
	}
	return sub
}

// splitTrimCSV mirrors `[e.strip() for e in raw.split(",")] if raw else []`.
// An empty string short-circuits to an empty (non-nil) slice.
func splitTrimCSV(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// LoadWebhookSubscriptions mirrors hermes_cli/webhook.py::_load_subscriptions:
// a missing file yields an empty map; a decode failure or a non-object JSON
// root is silently coerced to an empty map (matching the upstream `data if
// isinstance(data, dict) else {}` clause). Only I/O errors other than
// os.ErrNotExist bubble up.
func LoadWebhookSubscriptions(path string) (map[string]WebhookSubscription, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]WebhookSubscription{}, nil
		}
		return nil, fmt.Errorf("read webhook subscriptions %q: %w", path, err)
	}
	var out map[string]WebhookSubscription
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]WebhookSubscription{}, nil
	}
	if out == nil {
		out = map[string]WebhookSubscription{}
	}
	return out, nil
}

// SaveWebhookSubscriptions writes `subs` atomically: marshal, write to a
// sibling `.tmp`, then rename into place. Mirrors
// hermes_cli/webhook.py::_save_subscriptions, including the mkdir-parents
// step for first-run installs. A nil map is persisted as an empty JSON
// object so downstream loaders see a stable shape.
func SaveWebhookSubscriptions(path string, subs map[string]WebhookSubscription) error {
	if subs == nil {
		subs = map[string]WebhookSubscription{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir webhook subscriptions dir: %w", err)
	}
	data, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal webhook subscriptions: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write webhook subscriptions tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename webhook subscriptions: %w", err)
	}
	return nil
}
