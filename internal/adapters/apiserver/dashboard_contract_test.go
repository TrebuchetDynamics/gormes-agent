package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestDashboardContract_CoversNativeDashboardEndpoints(t *testing.T) {
	loop := &dashboardContractLoop{
		result: TurnResult{
			Content:   "native answer",
			SessionID: "sess-dashboard",
			Usage:     Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
			Messages:  []ChatMessage{{Role: "assistant", Content: "native answer"}},
		},
		streamTokens: []string{"native ", "stream"},
		toolProgress: []ToolProgressEvent{{
			Name:    "repo_search",
			Preview: "scanning internal/apiserver",
			Status:  "running",
		}},
	}
	srv := NewServer(Config{
		ModelName:             "gormes-agent",
		DashboardSessionToken: "fixture-token",
		ProviderName:          "native",
		Loop:                  loop,
		ResponseStore:         NewResponseStore(10),
		ModelProviders: []DashboardModelProvider{{
			Name:        "Native Gormes",
			Slug:        "native",
			Models:      []string{"gormes-agent"},
			TotalModels: 1,
			IsCurrent:   true,
		}},
		OAuthProviders: []DashboardOAuthProvider{{
			ID:         "anthropic",
			Name:       "Anthropic",
			Flow:       "external",
			CLICommand: "gormes auth add anthropic",
			DocsURL:    "https://docs.anthropic.com",
			Status: DashboardOAuthStatus{
				LoggedIn: false,
				Error:    "not_configured",
			},
		}},
	})
	h := srv.Handler()
	dashboardAuth := map[string]string{"X-Hermes-Session-Token": "fixture-token"}

	chat := postJSON(t, h, "/v1/chat/completions", map[string]any{
		"model":    "gormes-agent",
		"messages": []any{map[string]any{"role": "user", "content": "hello"}},
	}, nil)
	if chat.Code != http.StatusOK {
		t.Fatalf("chat status = %d, want 200; body=%s", chat.Code, chat.Body.String())
	}
	if got := chat.Header().Get("X-Hermes-Session-Id"); got != "sess-dashboard" {
		t.Fatalf("chat session header = %q, want sess-dashboard", got)
	}

	stream := postJSON(t, h, "/v1/chat/completions", map[string]any{
		"model":    "gormes-agent",
		"stream":   true,
		"messages": []any{map[string]any{"role": "user", "content": "stream please"}},
	}, map[string]string{"X-Hermes-Session-Id": "sess-dashboard"})
	if stream.Code != http.StatusOK {
		t.Fatalf("stream status = %d, want 200; body=%s", stream.Code, stream.Body.String())
	}
	if got := stream.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("stream Content-Type = %q, want text/event-stream", got)
	}
	for _, want := range []string{`"object":"chat.completion.chunk"`, `"content":"native "`, "data: [DONE]"} {
		if !strings.Contains(stream.Body.String(), want) {
			t.Fatalf("stream body missing %q: %s", want, stream.Body.String())
		}
	}

	response := postJSON(t, h, "/v1/responses", map[string]any{
		"model": "gormes-agent",
		"input": "persist this turn",
	}, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var created ResponseObject
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("response ID is empty")
	}

	sessions := getJSON(t, h, "/api/sessions?limit=10&offset=0", dashboardAuth)
	if sessions.Code != http.StatusOK {
		t.Fatalf("sessions status = %d, want 200; body=%s", sessions.Code, sessions.Body.String())
	}
	var sessionList struct {
		Sessions []struct {
			ID           string  `json:"id"`
			Model        *string `json:"model"`
			MessageCount int     `json:"message_count"`
			Preview      *string `json:"preview"`
		} `json:"sessions"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := json.Unmarshal(sessions.Body.Bytes(), &sessionList); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if sessionList.Total != 1 || len(sessionList.Sessions) != 1 {
		t.Fatalf("sessions = %+v, want one native session", sessionList)
	}
	if got := sessionList.Sessions[0].ID; got != "sess-dashboard" {
		t.Fatalf("session id = %q, want sess-dashboard", got)
	}
	if sessionList.Sessions[0].MessageCount == 0 || sessionList.Sessions[0].Preview == nil || *sessionList.Sessions[0].Preview == "" {
		t.Fatalf("session summary missing message count or preview: %+v", sessionList.Sessions[0])
	}

	modelOptions := getJSON(t, h, "/api/model/options", dashboardAuth)
	if modelOptions.Code != http.StatusOK {
		t.Fatalf("model options status = %d, want 200; body=%s", modelOptions.Code, modelOptions.Body.String())
	}
	var options struct {
		Model     string `json:"model"`
		Provider  string `json:"provider"`
		Providers []struct {
			Name      string   `json:"name"`
			Slug      string   `json:"slug"`
			Models    []string `json:"models"`
			IsCurrent bool     `json:"is_current"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(modelOptions.Body.Bytes(), &options); err != nil {
		t.Fatalf("decode model options: %v", err)
	}
	if options.Model != "gormes-agent" || options.Provider != "native" || len(options.Providers) != 1 || !options.Providers[0].IsCurrent {
		t.Fatalf("model options = %+v, want current native provider", options)
	}

	oauth := getJSON(t, h, "/api/providers/oauth", dashboardAuth)
	if oauth.Code != http.StatusOK {
		t.Fatalf("oauth status = %d, want 200; body=%s", oauth.Code, oauth.Body.String())
	}
	var oauthStatus struct {
		Providers []struct {
			ID     string `json:"id"`
			Flow   string `json:"flow"`
			Status struct {
				LoggedIn bool   `json:"logged_in"`
				Error    string `json:"error"`
			} `json:"status"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(oauth.Body.Bytes(), &oauthStatus); err != nil {
		t.Fatalf("decode oauth: %v", err)
	}
	if len(oauthStatus.Providers) != 1 || oauthStatus.Providers[0].ID != "anthropic" || oauthStatus.Providers[0].Status.LoggedIn {
		t.Fatalf("oauth providers = %+v, want configured disconnected provider", oauthStatus.Providers)
	}
	if oauthStatus.Providers[0].Status.Error != "not_configured" {
		t.Fatalf("oauth error = %q, want not_configured", oauthStatus.Providers[0].Status.Error)
	}

	run := postJSON(t, h, "/v1/runs", map[string]any{"input": "show tool progress"}, nil)
	if run.Code != http.StatusAccepted {
		t.Fatalf("run status = %d, want 202; body=%s", run.Code, run.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(run.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	events := getJSON(t, h, "/v1/runs/"+started.RunID+"/events", nil)
	if events.Code != http.StatusOK {
		t.Fatalf("events status = %d, want 200; body=%s", events.Code, events.Body.String())
	}
	for _, want := range []string{`"event":"tool.progress"`, `"name":"repo_search"`, `"preview":"scanning internal/apiserver"`} {
		if !strings.Contains(events.Body.String(), want) {
			t.Fatalf("tool progress stream missing %q: %s", want, events.Body.String())
		}
	}

	deleted := deleteJSON(t, h, "/api/sessions/sess-dashboard", dashboardAuth)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete session status = %d, want 200; body=%s", deleted.Code, deleted.Body.String())
	}
	var deleteBody struct {
		OK        bool   `json:"ok"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(deleted.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("decode delete session: %v", err)
	}
	if !deleteBody.OK || deleteBody.SessionID != "sess-dashboard" {
		t.Fatalf("delete body = %+v, want ok for sess-dashboard", deleteBody)
	}
	missing := getJSON(t, h, "/v1/responses/"+created.ID, nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("response after session delete status = %d, want 404; body=%s", missing.Code, missing.Body.String())
	}
}

// TestDashboard_BuildAttributionPluginsAndDelete proves the remaining
// authenticated dashboard endpoints — `/api/plugins` and the session
// DELETE response — also embed BuildInfo. Slices 110-112 covered
// /api/status, kanban, model, oauth, and sessions; this rounds out
// every JSON-emitting dashboard endpoint with the same attribution
// contract.
func TestDashboard_BuildAttributionPluginsAndDelete(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-build-rest",
			GitCommit: "fee1b00d",
			GitDirty:  true,
			GoVersion: "go1.23.0-test",
		},
	})
	h := srv.Handler()

	t.Run("plugins", func(t *testing.T) {
		rec := getJSON(t, h, "/api/dashboard/plugins", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("/api/plugins status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
				GitDirty  bool   `json:"git_dirty"`
			} `json:"build"`
			Runtime struct {
				State string `json:"state"`
			} `json:"runtime"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
		}
		if got.Build.Version != "test-build-rest" || got.Build.GitCommit != "fee1b00d" || !got.Build.GitDirty {
			t.Errorf("build = %+v, want version=test-build-rest commit=fee1b00d dirty=true", got.Build)
		}
		if got.Runtime.State == "" {
			t.Errorf("runtime.state empty — top-level fields lost via wrapping")
		}
	})

	t.Run("session_delete", func(t *testing.T) {
		store := NewResponseStore(10)
		seeded := NewServer(Config{
			ModelName:     "gormes-agent",
			ResponseStore: store,
			BuildInfo: BuildInfo{
				Version:   "test-build-rest",
				GitCommit: "fee1b00d",
				GitDirty:  true,
				GoVersion: "go1.23.0-test",
			},
		})
		if err := store.Put("resp_seed", StoredResponse{
			Response:  ResponseObject{ID: "resp_seed", Object: "response", Status: "completed", Model: "gormes-agent"},
			SessionID: "sess-delete",
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
		rec := deleteJSON(t, seeded.Handler(), "/api/sessions/sess-delete", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete status = %d; body=%s", rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
				GitDirty  bool   `json:"git_dirty"`
			} `json:"build"`
			OK        bool   `json:"ok"`
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
		}
		if got.Build.Version != "test-build-rest" || got.Build.GitCommit != "fee1b00d" || !got.Build.GitDirty {
			t.Errorf("build = %+v, want version=test-build-rest commit=fee1b00d dirty=true", got.Build)
		}
		if !got.OK || got.SessionID != "sess-delete" {
			t.Errorf("body = %+v, want ok=true session_id=sess-delete", got)
		}
	})
}

// TestDashboard_BuildAttributionAcrossEndpoints proves the remaining
// authenticated dashboard endpoints carry the configured BuildInfo at
// the top of their JSON response so fleet automation aggregating
// dashboard state across machines can attribute every response to the
// binary version that emitted it. /api/status (slice 110), the kanban
// endpoints (slice 111), and these endpoints all use the same `build`
// envelope — a single source of truth across the dashboard surface.
func TestDashboard_BuildAttributionAcrossEndpoints(t *testing.T) {
	srv := NewServer(Config{
		ModelName:    "gormes-agent",
		ProviderName: "test-provider",
		BuildInfo: BuildInfo{
			Version:   "test-build-attr",
			GitCommit: "abc1234",
			GitDirty:  false,
			GoVersion: "go1.23.0-test",
		},
	})
	h := srv.Handler()

	for _, path := range []string{
		"/api/model/info",
		"/api/model/options",
		"/api/providers/oauth",
		"/api/sessions",
	} {
		rec := getJSON(t, h, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d; body=%s", path, rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
			} `json:"build"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: decode: %v\nbody=%s", path, err, rec.Body.String())
		}
		if got.Build.Version != "test-build-attr" {
			t.Errorf("%s: build.version = %q, want test-build-attr (body=%s)", path, got.Build.Version, rec.Body.String())
		}
		if got.Build.GitCommit != "abc1234" {
			t.Errorf("%s: build.git_commit = %q, want abc1234", path, got.Build.GitCommit)
		}
	}
}

// TestDashboardStatus_BuildAttribution proves the dashboard /api/status
// response carries the configured build version/git commit/dirty
// flag/Go version so fleet automation querying dashboards across
// machines can attribute each status response to the binary version
// that emitted it. Same convention as the rest of the `--json` arc.
// The configured BuildInfo (zero-value defaults to safe placeholders)
// must round-trip through `build` at the top of the JSON document.
func TestDashboardStatus_BuildAttribution(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-version-1.2.3",
			GitCommit: "deadbeef",
			GitDirty:  true,
			GoVersion: "go1.23.0-test",
		},
	})
	status := getJSON(t, srv.Handler(), "/api/status", nil)
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", status.Code, status.Body.String())
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
			GitDirty  bool   `json:"git_dirty"`
			GoVersion string `json:"go_version"`
		} `json:"build"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if got.Build.Version != "test-version-1.2.3" {
		t.Errorf("build.version = %q, want test-version-1.2.3", got.Build.Version)
	}
	if got.Build.GitCommit != "deadbeef" {
		t.Errorf("build.git_commit = %q, want deadbeef", got.Build.GitCommit)
	}
	if !got.Build.GitDirty {
		t.Errorf("build.git_dirty = false, want true")
	}
	if got.Build.GoVersion != "go1.23.0-test" {
		t.Errorf("build.go_version = %q, want go1.23.0-test", got.Build.GoVersion)
	}
}

func TestDashboardStatus_DegradesMissingNativeAndOptionalPanels(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent"})
	status := getJSON(t, srv.Handler(), "/api/status", nil)
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", status.Code, status.Body.String())
	}
	var got struct {
		Panels map[string]struct {
			State     string   `json:"state"`
			Reason    string   `json:"reason"`
			Endpoints []string `json:"endpoints"`
		} `json:"panels"`
		UpstreamReactRuntime struct {
			State    string `json:"state"`
			Required bool   `json:"required"`
		} `json:"upstream_react_runtime"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if got.Panels["chat"].State != "disabled" || !strings.Contains(got.Panels["chat"].Reason, "turn loop") {
		t.Fatalf("chat panel = %+v, want disabled turn-loop degradation", got.Panels["chat"])
	}
	if got.Panels["oauth"].State != "disabled" || got.Panels["plugins"].State != "disabled" {
		t.Fatalf("optional panel states = oauth:%+v plugins:%+v, want disabled", got.Panels["oauth"], got.Panels["plugins"])
	}
	if got.UpstreamReactRuntime.Required || got.UpstreamReactRuntime.State != "absent" {
		t.Fatalf("upstream runtime = %+v, want absent and not required", got.UpstreamReactRuntime)
	}
}

func TestDashboardContract_RejectsNonLoopbackHostHeaders(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", DashboardBoundHost: "127.0.0.1"})
	h := srv.Handler()

	allowed := getJSONWithHost(t, h, "/api/status", "localhost:9119", nil)
	if allowed.Code != http.StatusOK {
		t.Fatalf("loopback host status = %d, want 200; body=%s", allowed.Code, allowed.Body.String())
	}

	rejected := getJSONWithHost(t, h, "/api/status", "evil.example:9119", nil)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("non-loopback host status = %d, want 400; body=%s", rejected.Code, rejected.Body.String())
	}
	if !strings.Contains(rejected.Body.String(), "Invalid Host header") {
		t.Fatalf("rejection body = %s, want invalid-host explanation", rejected.Body.String())
	}
}

func TestDashboardContract_AllowsExplicitAllInterfacesHostMode(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", DashboardBoundHost: "0.0.0.0"})
	got := getJSONWithHost(t, srv.Handler(), "/api/status", "operator.example:9119", nil)
	if got.Code != http.StatusOK {
		t.Fatalf("all-interfaces host status = %d, want 200; body=%s", got.Code, got.Body.String())
	}
}

func TestDashboardContract_RequiresSessionTokenForSensitiveDashboardAPI(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", DashboardSessionToken: "fixture-token"})
	h := srv.Handler()

	public := getJSON(t, h, "/api/status", nil)
	if public.Code != http.StatusOK {
		t.Fatalf("public status code = %d, want 200; body=%s", public.Code, public.Body.String())
	}

	unauthorized := getJSON(t, h, "/api/model/options", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("missing session token status = %d, want 401; body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	if !strings.Contains(unauthorized.Body.String(), "Unauthorized") {
		t.Fatalf("missing session token body = %s, want Unauthorized", unauthorized.Body.String())
	}

	headerAuthorized := getJSON(t, h, "/api/model/options", map[string]string{"X-Hermes-Session-Token": "fixture-token"})
	if headerAuthorized.Code != http.StatusOK {
		t.Fatalf("session-header status = %d, want 200; body=%s", headerAuthorized.Code, headerAuthorized.Body.String())
	}

	bearerAuthorized := getJSON(t, h, "/api/providers/oauth", map[string]string{"Authorization": "Bearer fixture-token"})
	if bearerAuthorized.Code != http.StatusOK {
		t.Fatalf("bearer-token status = %d, want 200; body=%s", bearerAuthorized.Code, bearerAuthorized.Body.String())
	}
}

func TestDashboardContract_DoesNotAddNodeOrReactRuntimeAssets(t *testing.T) {
	for _, path := range []string{"package.json", "vite.config.ts", "node_modules"} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("internal/apiserver unexpectedly contains upstream runtime asset %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
	tsFiles, err := filepath.Glob("*.ts")
	if err != nil {
		t.Fatalf("glob TypeScript files: %v", err)
	}
	if len(tsFiles) != 0 {
		t.Fatalf("internal/apiserver TypeScript files = %v, want none", tsFiles)
	}
}

func getJSONWithHost(t *testing.T, h http.Handler, path string, host string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = host
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	h.ServeHTTP(rec, req)
	_, _ = io.Copy(io.Discard, rec.Result().Body)
	return rec
}

type dashboardContractLoop struct {
	mu           sync.Mutex
	calls        []TurnRequest
	result       TurnResult
	streamTokens []string
	toolProgress []ToolProgressEvent
}

func (d *dashboardContractLoop) RunTurn(_ context.Context, req TurnRequest) (TurnResult, error) {
	d.mu.Lock()
	d.calls = append(d.calls, req)
	d.mu.Unlock()
	return d.result, nil
}

func (d *dashboardContractLoop) StreamTurn(_ context.Context, req TurnRequest, cb StreamCallbacks) (TurnResult, error) {
	d.mu.Lock()
	d.calls = append(d.calls, req)
	d.mu.Unlock()
	for _, ev := range d.toolProgress {
		if cb.OnToolProgress != nil {
			if err := cb.OnToolProgress(ev); err != nil {
				return TurnResult{}, err
			}
		}
	}
	for _, token := range d.streamTokens {
		if cb.OnToken != nil {
			if err := cb.OnToken(token); err != nil {
				return TurnResult{}, err
			}
		}
	}
	return d.result, nil
}
