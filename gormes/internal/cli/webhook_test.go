package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tests freeze the Go port of the deterministic, dependency-free helpers in
// hermes_cli/webhook.py (Phase 5.O). The interactive CLI subcommand (argparse
// wiring, stdout printing, HTTP test POST) stays intentionally unported — the
// webhook adapter's hot-reload contract is the only cross-implementation
// contract worth pinning, and it lives entirely in these pure primitives.

func TestNormalizeWebhookName_HappyPathsMatchUpstream(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"already canonical", "github-alerts", "github-alerts"},
		{"uppercase collapsed", "GitHubAlerts", "githubalerts"},
		{"leading/trailing whitespace stripped", "  github-alerts  ", "github-alerts"},
		{"single-space word collapsed to hyphen", "github alerts", "github-alerts"},
		{"multi-space words each become hyphens", "my github alerts", "my-github-alerts"},
		{"underscore preserved", "github_alerts", "github_alerts"},
		{"leading digit accepted", "7stream", "7stream"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeWebhookName(tc.in)
			if err != nil {
				t.Fatalf("NormalizeWebhookName(%q) error = %v, want nil", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeWebhookName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeWebhookName_InvalidNamesReturnSentinel(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"leading hyphen", "-foo"},
		{"leading underscore", "_foo"},
		{"slash path segment", "foo/bar"},
		{"dot segment", "foo.bar"},
		{"non-ascii symbol", "foo$bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeWebhookName(tc.in)
			if err == nil {
				t.Fatalf("NormalizeWebhookName(%q) = %q, want error", tc.in, got)
			}
			if !errors.Is(err, ErrInvalidWebhookName) {
				t.Fatalf("NormalizeWebhookName(%q) error = %v, want ErrInvalidWebhookName", tc.in, err)
			}
			if got != "" {
				t.Fatalf("NormalizeWebhookName(%q) = %q, want empty on error", tc.in, got)
			}
		})
	}
}

func TestFormatWebhookBaseURL_WildcardHostBecomesLocalhost(t *testing.T) {
	got := FormatWebhookBaseURL("0.0.0.0", 8644)
	want := "http://localhost:8644"
	if got != want {
		t.Fatalf("FormatWebhookBaseURL(\"0.0.0.0\", 8644) = %q, want %q", got, want)
	}
}

func TestFormatWebhookBaseURL_EmptyHostFallsBackToLocalhost(t *testing.T) {
	got := FormatWebhookBaseURL("", 8644)
	want := "http://localhost:8644"
	if got != want {
		t.Fatalf("FormatWebhookBaseURL(\"\", 8644) = %q, want %q", got, want)
	}
}

func TestFormatWebhookBaseURL_CustomHostPreserved(t *testing.T) {
	got := FormatWebhookBaseURL("hooks.example.com", 8080)
	want := "http://hooks.example.com:8080"
	if got != want {
		t.Fatalf("FormatWebhookBaseURL(\"hooks.example.com\", 8080) = %q, want %q", got, want)
	}
}

func TestFormatWebhookBaseURL_ZeroPortFallsBackToDefault(t *testing.T) {
	got := FormatWebhookBaseURL("localhost", 0)
	want := "http://localhost:8644"
	if got != want {
		t.Fatalf("FormatWebhookBaseURL(\"localhost\", 0) = %q, want %q", got, want)
	}
}

func TestBuildWebhookSubscription_FillsDefaults(t *testing.T) {
	now := time.Date(2026, 4, 24, 7, 36, 45, 0, time.UTC)
	got := BuildWebhookSubscription(WebhookSubscriptionOpts{
		Name:   "alerts",
		Secret: "supersecret",
	}, now)

	if got.Description != "Agent-created subscription: alerts" {
		t.Fatalf("default description = %q, want %q", got.Description, "Agent-created subscription: alerts")
	}
	if got.Deliver != "log" {
		t.Fatalf("default deliver = %q, want %q", got.Deliver, "log")
	}
	if got.Secret != "supersecret" {
		t.Fatalf("secret passthrough = %q, want %q", got.Secret, "supersecret")
	}
	if got.CreatedAt != "2026-04-24T07:36:45Z" {
		t.Fatalf("CreatedAt = %q, want %q", got.CreatedAt, "2026-04-24T07:36:45Z")
	}
	if got.Events == nil {
		t.Fatalf("Events = nil, want empty slice so JSON stays [] not null")
	}
	if len(got.Events) != 0 {
		t.Fatalf("Events = %v, want empty", got.Events)
	}
	if got.Skills == nil {
		t.Fatalf("Skills = nil, want empty slice so JSON stays [] not null")
	}
	if len(got.Skills) != 0 {
		t.Fatalf("Skills = %v, want empty", got.Skills)
	}
	if got.DeliverExtra != nil {
		t.Fatalf("DeliverExtra = %v, want nil when DeliverChatID empty", got.DeliverExtra)
	}
}

func TestBuildWebhookSubscription_SplitsAndTrimsCSVs(t *testing.T) {
	now := time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC)
	got := BuildWebhookSubscription(WebhookSubscriptionOpts{
		Name:        "ci",
		Description: "CI hook",
		Events:      "push, pull_request , workflow_run",
		Skills:      "gh-summary, ci-triage",
		Secret:      "s",
		Prompt:      "Triage the incoming event.",
		Deliver:     "slack",
	}, now)

	wantEvents := []string{"push", "pull_request", "workflow_run"}
	if !reflect.DeepEqual(got.Events, wantEvents) {
		t.Fatalf("Events = %v, want %v", got.Events, wantEvents)
	}
	wantSkills := []string{"gh-summary", "ci-triage"}
	if !reflect.DeepEqual(got.Skills, wantSkills) {
		t.Fatalf("Skills = %v, want %v", got.Skills, wantSkills)
	}
	if got.Description != "CI hook" {
		t.Fatalf("Description = %q, want %q", got.Description, "CI hook")
	}
	if got.Deliver != "slack" {
		t.Fatalf("Deliver = %q, want %q", got.Deliver, "slack")
	}
	if got.Prompt != "Triage the incoming event." {
		t.Fatalf("Prompt = %q, want %q", got.Prompt, "Triage the incoming event.")
	}
}

func TestBuildWebhookSubscription_DeliverChatIDPopulatesExtra(t *testing.T) {
	now := time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC)
	got := BuildWebhookSubscription(WebhookSubscriptionOpts{
		Name:          "alerts",
		Secret:        "s",
		DeliverChatID: "-100123456",
	}, now)

	if got.DeliverExtra == nil {
		t.Fatalf("DeliverExtra = nil, want populated map")
	}
	if got.DeliverExtra["chat_id"] != "-100123456" {
		t.Fatalf("DeliverExtra[chat_id] = %v, want %q", got.DeliverExtra["chat_id"], "-100123456")
	}
}

func TestBuildWebhookSubscription_CreatedAtIsUTCZuluFormat(t *testing.T) {
	// A non-UTC instant must still be serialized with a trailing Z, mirroring
	// upstream's time.gmtime+%Y-%m-%dT%H:%M:%SZ format.
	loc := time.FixedZone("UTC+2", 2*3600)
	local := time.Date(2026, 4, 24, 9, 36, 45, 0, loc)
	got := BuildWebhookSubscription(WebhookSubscriptionOpts{
		Name:   "alerts",
		Secret: "s",
	}, local)

	if got.CreatedAt != "2026-04-24T07:36:45Z" {
		t.Fatalf("CreatedAt = %q, want %q", got.CreatedAt, "2026-04-24T07:36:45Z")
	}
}

func TestBuildWebhookSubscription_JSONOmitsDeliverExtraWhenUnset(t *testing.T) {
	// Upstream only writes "deliver_extra" when args.deliver_chat_id is set.
	// The Go port must mirror that — otherwise the gateway's hot-reload diff
	// would treat every subscription as modified.
	now := time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC)
	sub := BuildWebhookSubscription(WebhookSubscriptionOpts{
		Name:   "alerts",
		Secret: "s",
	}, now)
	data, err := json.Marshal(sub)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "deliver_extra") {
		t.Fatalf("marshaled subscription %q unexpectedly contains deliver_extra", data)
	}
}

func TestLoadWebhookSubscriptions_MissingFileReturnsEmptyMap(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "webhook_subscriptions.json")
	got, err := LoadWebhookSubscriptions(path)
	if err != nil {
		t.Fatalf("LoadWebhookSubscriptions(missing) error = %v, want nil", err)
	}
	if got == nil {
		t.Fatalf("LoadWebhookSubscriptions(missing) = nil, want non-nil empty map")
	}
	if len(got) != 0 {
		t.Fatalf("LoadWebhookSubscriptions(missing) = %v, want empty map", got)
	}
}

func TestLoadWebhookSubscriptions_MalformedJSONReturnsEmptyMap(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "webhook_subscriptions.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := LoadWebhookSubscriptions(path)
	if err != nil {
		t.Fatalf("LoadWebhookSubscriptions(malformed) error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadWebhookSubscriptions(malformed) = %v, want empty map", got)
	}
}

func TestLoadWebhookSubscriptions_NonObjectRootReturnsEmptyMap(t *testing.T) {
	// Upstream swallows `data if isinstance(data, dict) else {}` — a bare list
	// or string at the root must not leak through as an opaque map.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "webhook_subscriptions.json")
	if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := LoadWebhookSubscriptions(path)
	if err != nil {
		t.Fatalf("LoadWebhookSubscriptions(list root) error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadWebhookSubscriptions(list root) = %v, want empty map", got)
	}
}

func TestSaveWebhookSubscriptions_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nested", "webhook_subscriptions.json")

	now := time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC)
	sub := BuildWebhookSubscription(WebhookSubscriptionOpts{
		Name:          "alerts",
		Description:   "CI alerts",
		Events:        "push, pull_request",
		Secret:        "s",
		Skills:        "triage",
		Deliver:       "slack",
		DeliverChatID: "C01",
	}, now)
	want := map[string]WebhookSubscription{"alerts": sub}

	if err := SaveWebhookSubscriptions(path, want); err != nil {
		t.Fatalf("SaveWebhookSubscriptions: %v", err)
	}

	got, err := LoadWebhookSubscriptions(path)
	if err != nil {
		t.Fatalf("LoadWebhookSubscriptions: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestSaveWebhookSubscriptions_AtomicRenameLeavesNoTempFile(t *testing.T) {
	// Mirrors upstream's write-to-.tmp + os.replace flow: the .tmp file must
	// not remain after a successful save.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "webhook_subscriptions.json")
	subs := map[string]WebhookSubscription{
		"alerts": {
			Description: "x",
			Events:      []string{},
			Skills:      []string{},
			Secret:      "s",
			Deliver:     "log",
			CreatedAt:   "2026-04-24T00:00:00Z",
		},
	}
	if err := SaveWebhookSubscriptions(path, subs); err != nil {
		t.Fatalf("SaveWebhookSubscriptions: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected final file at %q, stat err: %v", path, err)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("leftover temp file at %q (err=%v), want not-exist", path+".tmp", err)
	}
}

func TestSaveWebhookSubscriptions_NilInputWritesEmptyObject(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "webhook_subscriptions.json")
	if err := SaveWebhookSubscriptions(path, nil); err != nil {
		t.Fatalf("SaveWebhookSubscriptions(nil): %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.TrimSpace(string(b)) != "{}" {
		t.Fatalf("on-disk contents = %q, want empty JSON object", b)
	}
}
