package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func detailedHealthFixtureInput() DetailedHealthSnapshotInput {
	return DetailedHealthSnapshotInput{
		Provider: DetailedHealthProviderInput{
			Name:              "openai",
			Model:             "gpt-4.1",
			Configured:        true,
			APIKey:            "plain-provider-key-placeholder",
			RawRequestPayload: `{"messages":[{"content":"raw request placeholder"}]}`,
		},
		ResponseStore: DetailedHealthResponseStoreInput{
			Enabled:      true,
			Stored:       3,
			MaxStored:    100,
			LRUEvictions: 1,
		},
		RunEvents: DetailedHealthRunEventsInput{
			Available:     true,
			Active:        2,
			OrphanedSwept: 1,
			TTLSeconds:    300,
		},
		Gateway: DetailedHealthGatewayInput{
			Available:    true,
			State:        "running",
			ActiveAgents: 1,
			Platforms:    map[string]string{"telegram": "running"},
			ProxyState:   "ready",
			Token:        "gateway-token-secret",
		},
		Cron: DetailedHealthCronInput{
			Available:     true,
			Enabled:       true,
			Jobs:          4,
			Paused:        1,
			LastRunStatus: "success",
			ScriptBodies:  []string{"curl https://example.invalid?token=cron-script-secret"},
		},
	}
}

func newDetailedHealthFixtureServer(t *testing.T, apiKey string) *Server {
	t.Helper()
	return NewServer(Config{
		APIKey:    apiKey,
		ModelName: "gormes-agent",
		Loop:      &fakeTurnLoop{},
		DetailedHealth: func() DetailedHealthSnapshotInput {
			return detailedHealthFixtureInput()
		},
	})
}

// TestAPIServerDetailedHealth_BuildAttribution proves
// `/health/detailed` and `/v1/health/detailed` carry the configured
// BuildInfo at the top of the JSON response so fleet automation
// running detailed health probes across machines can attribute every
// response to the binary version that emitted it. Same convention as
// `/health` (slice 117) and the rest of the API server JSON arc.
func TestAPIServerDetailedHealth_BuildAttribution(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-detailed-health",
			GitCommit: "ba5eba11",
		},
	})
	for _, path := range []string{"/health/detailed", "/v1/health/detailed"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d; body=%s", path, rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
			} `json:"build"`
			Provider struct {
				Status string `json:"status"`
			} `json:"provider"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s decode: %v\nbody=%s", path, err, rec.Body.String())
		}
		if got.Build.Version != "test-detailed-health" {
			t.Errorf("%s build.version = %q, want test-detailed-health (body=%s)", path, got.Build.Version, rec.Body.String())
		}
		if got.Build.GitCommit != "ba5eba11" {
			t.Errorf("%s build.git_commit = %q, want ba5eba11", path, got.Build.GitCommit)
		}
		if got.Provider.Status == "" {
			t.Errorf("%s provider.status empty — top-level fields lost", path)
		}
	}
}

func TestAPIServerDetailedHealthEndpoint_OK(t *testing.T) {
	srv := newDetailedHealthFixtureServer(t, "")

	for _, path := range []string{"/health/detailed", "/v1/health/detailed"} {
		path := path
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil {
				t.Fatalf("decode body: %v; body=%s", err, rec.Body.String())
			}
			for _, key := range []string{"provider", "response_store", "run_events", "gateway", "cron"} {
				if _, ok := fields[key]; !ok {
					t.Fatalf("response missing %q section: %s", key, rec.Body.String())
				}
			}
		})
	}
}

func TestAPIServerDetailedHealthEndpoint_NoAuthRequired(t *testing.T) {
	srv := newDetailedHealthFixtureServer(t, "plain-api-key-placeholder")

	for _, path := range []string{"/health/detailed", "/v1/health/detailed"} {
		path := path
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 without auth; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAPIServerDetailedHealthEndpoint_MethodNotAllowed(t *testing.T) {
	srv := newDetailedHealthFixtureServer(t, "")

	for _, path := range []string{"/health/detailed", "/v1/health/detailed"} {
		path := path
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, nil)
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 405; body=%s", rec.Code, rec.Body.String())
			}
			var envelope struct {
				Error struct {
					Type    string `json:"type"`
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v; body=%s", err, rec.Body.String())
			}
			if envelope.Error.Type != "invalid_request_error" {
				t.Fatalf("error.type = %q, want invalid_request_error", envelope.Error.Type)
			}
			if envelope.Error.Code != "method_not_allowed" {
				t.Fatalf("error.code = %q, want method_not_allowed", envelope.Error.Code)
			}
		})
	}
}

func TestAPIServerDetailedHealthEndpoint_RedactsSecrets(t *testing.T) {
	srv := newDetailedHealthFixtureServer(t, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, secret := range []string{
		"plain-provider-key-placeholder",
		"raw request placeholder",
		"gateway-token-secret",
		"cron-script-secret",
		"curl https://example.invalid",
	} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked secret %q: %s", secret, body)
		}
	}
}

func TestAPIServerDetailedHealthEndpoint_FlatHealthUnchanged(t *testing.T) {
	srv := newDetailedHealthFixtureServer(t, "")

	for _, path := range []string{"/health", "/v1/health"} {
		path := path
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode flat health: %v; body=%s", err, rec.Body.String())
			}
			for _, key := range []string{"status", "platform", "responses", "runs"} {
				if _, ok := got[key]; !ok {
					t.Fatalf("flat health missing key %q: %s", key, rec.Body.String())
				}
			}
			if _, ok := got["provider"]; ok {
				t.Fatalf("flat health unexpectedly carries provider section: %s", rec.Body.String())
			}
		})
	}
}
