package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestUsageCommand_HTTPClientHasBoundedTimeout proves the package-level
// HTTP client carries a non-zero Timeout so an unresponsive provider
// can't hang `gormes usage` indefinitely. Same defensive bound as
// logsHTTPClient (slice 40); without it, http.DefaultClient's lack of
// timeout leaks through to operator terminals.
func TestUsageCommand_HTTPClientHasBoundedTimeout(t *testing.T) {
	if usageHTTPClient == nil {
		t.Fatal("usageHTTPClient must be configured at package init")
	}
	if usageHTTPClient.Timeout <= 0 {
		t.Fatalf("usageHTTPClient.Timeout = %s, want a positive bound", usageHTTPClient.Timeout)
	}
	// Sanity bound: must be tighter than what an operator will tolerate.
	// 60s is generous but bounded; >60s defeats the point.
	if usageHTTPClient.Timeout > 60*time.Second {
		t.Fatalf("usageHTTPClient.Timeout = %s, want <= 60s for operator responsiveness", usageHTTPClient.Timeout)
	}
}

func TestUsageCommand_RendersUnsupportedProviderWithoutStartingTUI(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "usage", "--provider", "fixture-provider")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{"Provider: fixture-provider", "Usage unavailable: account usage is not supported for provider fixture-provider"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
	if strings.Contains(stderr, "api_server") {
		t.Fatalf("stderr contains api_server health output:\n%s", stderr)
	}
}
func TestUsageCommand_InfersProviderFromConfiguredModel(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	var sawAuthorization bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("request path = %q, want /wham/usage", r.URL.Path)
		}
		sawAuthorization = r.Header.Get("Authorization") == "Bearer fixture-token"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type":"pro",
			"rate_limit":{
				"primary_window":{"used_percent":25},
				"secondary_window":{"used_percent":50}
			}
		}`))
	}))
	defer server.Close()
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "`+server.URL+`"
api_key = "fixture-token"
model = "gpt-5.5"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "usage")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"Provider: openai-codex (Pro)",
		"Session: 75% remaining (25% used)",
		"Weekly: 50% remaining (50% used)",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
	if !sawAuthorization {
		t.Fatalf("account usage request did not use configured API key")
	}
}

func TestUsageCommand_InfersProviderFromEnvModel(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INFERENCE_MODEL", "claude-sonnet-4-20250514")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oauth/usage" {
			t.Fatalf("request path = %q, want /api/oauth/usage", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"five_hour":{"utilization":0.2},
			"seven_day":{"utilization":0.4}
		}`))
	}))
	defer server.Close()
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "`+server.URL+`"
api_key = "oauth-token"
model = "hermes-agent"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "usage")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"Provider: anthropic",
		"5-hour: 80% remaining (20% used)",
		"7-day: 60% remaining (40% used)",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
}

// TestUsageCommand_JSONEmitsStructuredSnapshot proves
// `gormes usage --json` returns a parseable
// `{build, provider, account_id, plan, source, fetched_at,
// windows: [...], details, unavailable}` document so fleet automation
// tracking provider quota across machines can plot dashboards
// without scraping the multi-line "Provider: X / Session: Y%..." prose.
func TestUsageCommand_JSONEmitsStructuredSnapshot(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type":"pro",
			"rate_limit":{
				"primary_window":{"used_percent":25},
				"secondary_window":{"used_percent":50}
			}
		}`))
	}))
	defer server.Close()
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "`+server.URL+`"
api_key = "fixture-token"
model = "gpt-5.5"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "usage", "--json")
	if err != nil {
		t.Fatalf("usage --json: %v\nstderr=%s", err, stderr)
	}
	// Raw API key MUST never leak into stdout.
	if strings.Contains(stdout+stderr, "fixture-token") {
		t.Fatalf("usage --json LEAKED the api key:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Provider string `json:"provider"`
		Plan     string `json:"plan,omitempty"`
		Windows  []struct {
			Label       string   `json:"label"`
			UsedPercent *float64 `json:"used_percent,omitempty"`
		} `json:"windows"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("usage --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Provider != "openai-codex" {
		t.Errorf("provider = %q, want openai-codex", got.Provider)
	}
	if len(got.Windows) < 2 {
		t.Errorf("windows len = %d, want >=2", len(got.Windows))
	}
}
