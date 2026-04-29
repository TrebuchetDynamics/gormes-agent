package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
