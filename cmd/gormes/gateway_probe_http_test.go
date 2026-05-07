package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestGatewayProbeCommandHTTPVerifiesCapabilitiesWithRedactedAuthSource(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	const secret = "sk-gateway-probe-secret"
	t.Setenv("GATEWAY_PROXY_KEY", secret)
	restoreRuntime := gatewayProbeRuntimeSummaryForTest(t, tools.GatewayRuntimeSummary{
		State:          "running",
		Validation:     "live",
		ValidationLive: true,
	})
	defer restoreRuntime()
	server := newGatewayProbeCommandHTTPFixture(t, secret, func(w http.ResponseWriter, _ *http.Request) {
		writeGatewayProbeCommandCapabilities(t, w, true)
	})
	defer server.Close()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "gateway", "probe", "--url", server.URL, "--json")
	if err != nil {
		t.Fatalf("gateway probe --url: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, stream := range []string{stdout, stderr} {
		if strings.Contains(stream, secret) {
			t.Fatalf("probe output leaked auth token:\nstdout=%s\nstderr=%s", stdout, stderr)
		}
	}
	var got tools.GatewayProbeResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if !got.OK || len(got.Targets) != 1 {
		t.Fatalf("probe result = %+v, want one verified target", got)
	}
	target := got.Targets[0]
	if target.Health != tools.GatewayHealthHTTPHealthy || target.Status != tools.GatewayProbeStatusCapabilityReady {
		t.Fatalf("target = %+v, want HTTP capability-ready target", target)
	}
	if target.Capabilities == nil || target.Capabilities.AuthSource != "env:GATEWAY_PROXY_KEY" || !target.Capabilities.AuthRequired {
		t.Fatalf("capabilities = %+v, want redacted env auth source", target.Capabilities)
	}
}

func TestGatewayProbeCommandHTTPClassifiesUnauthorizedAndMalformedCapabilities(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	restoreRuntime := gatewayProbeRuntimeSummaryForTest(t, tools.GatewayRuntimeSummary{State: "running"})
	defer restoreRuntime()

	t.Run("unauthorized", func(t *testing.T) {
		server := newGatewayProbeCommandHTTPFixture(t, "sk-required", func(w http.ResponseWriter, _ *http.Request) {
			writeGatewayProbeCommandCapabilities(t, w, true)
		})
		defer server.Close()

		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeOneshotFlagCommand(cmd, "gateway", "probe", "--url", server.URL, "--json")
		if err == nil {
			t.Fatalf("gateway probe --url unexpectedly succeeded\nstdout=%s\nstderr=%s", stdout, stderr)
		}
		if strings.Contains(stdout+stderr, "sk-required") {
			t.Fatalf("probe output leaked required token:\nstdout=%s\nstderr=%s", stdout, stderr)
		}
		var got tools.GatewayProbeResult
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, stdout)
		}
		target := got.Targets[0]
		if target.Health != tools.GatewayHealthHTTPUnauthorized || target.Status != tools.GatewayProbeStatusUnauthorized {
			t.Fatalf("target = %+v, want unauthorized HTTP classification", target)
		}
		if target.Capabilities == nil || target.Capabilities.AuthSource != "none" {
			t.Fatalf("capabilities = %+v, want auth_source none", target.Capabilities)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		server := newGatewayProbeCommandHTTPFixture(t, "", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":`))
		})
		defer server.Close()

		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeOneshotFlagCommand(cmd, "gateway", "probe", "--url", server.URL, "--json")
		if err == nil {
			t.Fatalf("gateway probe --url unexpectedly succeeded\nstdout=%s\nstderr=%s", stdout, stderr)
		}
		var got tools.GatewayProbeResult
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, stdout)
		}
		target := got.Targets[0]
		if target.Health != tools.GatewayHealthHTTPCapabilityMalformed || target.Status != tools.GatewayProbeStatusMalformedCapabilities {
			t.Fatalf("target = %+v, want malformed HTTP classification", target)
		}
	})
}

func newGatewayProbeCommandHTTPFixture(t *testing.T, secret string, capabilities http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			writeGatewayProbeCommandJSON(t, w, http.StatusOK, map[string]any{"status": "ok", "platform": "gormes-agent"})
		case "/health/detailed":
			writeGatewayProbeCommandJSON(t, w, http.StatusOK, map[string]any{"status": "ok", "platform": "gormes-agent"})
		case "/v1/capabilities":
			if secret != "" && r.Header.Get("Authorization") != "Bearer "+secret {
				writeGatewayProbeCommandJSON(t, w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"code": "invalid_api_key"}})
				return
			}
			capabilities(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func writeGatewayProbeCommandCapabilities(t *testing.T, w http.ResponseWriter, supported bool) {
	t.Helper()
	writeGatewayProbeCommandJSON(t, w, http.StatusOK, map[string]any{
		"object":   "hermes.api_server.capabilities",
		"platform": "gormes-agent",
		"model":    "gormes-agent",
		"auth":     map[string]any{"type": "bearer", "required": true},
		"features": map[string]any{
			"chat_completions":           true,
			"chat_completions_streaming": true,
			"responses_api":              true,
			"responses_streaming":        true,
			"run_submission":             true,
			"run_status":                 supported,
			"run_events_sse":             supported,
			"run_stop":                   true,
			"tool_progress_events":       true,
			"session_continuity_header":  "X-Hermes-Session-Id",
		},
		"endpoints": map[string]any{
			"health":           map[string]string{"method": "GET", "path": "/health"},
			"health_detailed":  map[string]string{"method": "GET", "path": "/health/detailed"},
			"chat_completions": map[string]string{"method": "POST", "path": "/v1/chat/completions"},
			"runs":             map[string]string{"method": "POST", "path": "/v1/runs"},
			"run_status":       map[string]string{"method": "GET", "path": "/v1/runs/{run_id}"},
			"run_events":       map[string]string{"method": "GET", "path": "/v1/runs/{run_id}/events"},
			"run_stop":         map[string]string{"method": "POST", "path": "/v1/runs/{run_id}/stop"},
		},
	})
}

func writeGatewayProbeCommandJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode fixture response: %v", err)
	}
}
