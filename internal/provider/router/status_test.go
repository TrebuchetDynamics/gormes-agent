package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRouterHealthReportsConfigWithoutSecrets(t *testing.T) {
	handler := NewHandler(routerServerFixtureConfig(), HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  fakeChatProvider{reply: "unused"},
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{"gormes-router", "configured", "route_count"} {
		if !strings.Contains(body, want) {
			t.Fatalf("healthz missing %q:\n%s", want, body)
		}
	}
	assertRouterStatusNoSecrets(t, body)
}

func TestRouterStatusCountersAndUsageAreRedacted(t *testing.T) {
	var logs bytes.Buffer
	provider := &sequenceChatProvider{results: []sequenceChatResult{
		{result: ChatCompletionResult{Content: "ok", Usage: &Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8}}},
		{err: errors.New("upstream 500 raw body sk-upstream-secret Authorization: Bearer router-local-secret")},
	}}
	handler := NewHandler(routerServerFixtureConfig(), HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  provider,
		Logf:      func(format string, args ...any) { logs.WriteString(routerLogLine(format, args...)) },
	})

	postRouterChat(t, handler, "primary-chat", "one")
	postRouterChatWantStatus(t, handler, "primary-chat", "two", http.StatusBadGateway)

	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	req.Header.Set("X-Api-Key", "router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var got struct {
		Status struct {
			State string `json:"state"`
		} `json:"status"`
		Counters struct {
			Attempts       int    `json:"attempts"`
			Successes      int    `json:"successes"`
			Failures       int    `json:"failures"`
			Fallbacks      int    `json:"fallbacks"`
			LastErrorClass string `json:"last_error_class"`
			Usage          Usage  `json:"usage"`
		} `json:"counters"`
		Routes []struct {
			Alias    string `json:"alias"`
			Status   string `json:"status"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("status JSON: %v\n%s", err, rr.Body.String())
	}
	if got.Status.State != "configured" || got.Counters.Attempts != 2 || got.Counters.Successes != 1 || got.Counters.Failures != 1 || got.Counters.Fallbacks != 0 || got.Counters.LastErrorClass != "upstream_request_failed" {
		t.Fatalf("status counters = %+v status=%+v", got.Counters, got.Status)
	}
	if got.Counters.Usage.PromptTokens != 3 || got.Counters.Usage.CompletionTokens != 5 || got.Counters.Usage.TotalTokens != 8 {
		t.Fatalf("usage = %+v, want 3/5/8", got.Counters.Usage)
	}
	if len(got.Routes) == 0 || got.Routes[0].Alias == "" || got.Routes[0].Provider == "" || got.Routes[0].Model == "" {
		t.Fatalf("routes missing redacted health data: %+v", got.Routes)
	}
	assertRouterStatusNoSecrets(t, rr.Body.String())
	assertRouterStatusNoSecrets(t, logs.String())
	if strings.Contains(logs.String(), "raw body") {
		t.Fatalf("logs leaked raw upstream error body:\n%s", logs.String())
	}
}

func TestRouterStatusAliasAndOptionalLocalUnavailable(t *testing.T) {
	cfg := routerServerFixtureConfig()
	cfg.Router.Routes = append(cfg.Router.Routes, config.RouterRouteCfg{
		Name:      "local-openai-compatible",
		Alias:     "local-chat",
		Provider:  "custom",
		Model:     "llama3.2",
		BaseURL:   "http://127.0.0.1:11434/v1",
		Transport: DefaultTransport,
		Optional:  true,
	})
	handler := NewHandler(cfg, HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  fakeChatProvider{reply: "unused"},
	})

	req := httptest.NewRequest(http.MethodGet, "/router/status", nil)
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "local-chat") || !strings.Contains(body, "unavailable") || !strings.Contains(body, "optional_local_route_unprobed_unavailable") {
		t.Fatalf("status did not report optional local unavailable evidence:\n%s", body)
	}
	assertRouterStatusNoSecrets(t, body)
}

func TestRouterStatusUsesFakeHealthProbe(t *testing.T) {
	cfg := routerServerFixtureConfig()
	cfg.Router.Routes = append(cfg.Router.Routes, config.RouterRouteCfg{
		Name:      "local-openai-compatible",
		Alias:     "local-chat",
		Provider:  "custom",
		Model:     "llama3.2",
		BaseURL:   "http://127.0.0.1:11434/v1",
		Transport: DefaultTransport,
		Optional:  true,
	})
	probeCalls := 0
	handler := NewHandler(cfg, HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  fakeChatProvider{reply: "unused"},
		Probe: func(context.Context, Route) ProbeResult {
			probeCalls++
			return ProbeResult{Available: true, Evidence: []string{"fake_probe_available"}}
		},
	})
	if probeCalls != 1 {
		t.Fatalf("probeCalls = %d, want exactly one bounded fake probe at handler construction", probeCalls)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "local-chat") || !strings.Contains(body, "available") || !strings.Contains(body, "fake_probe_available") {
		t.Fatalf("status did not expose fake probe health evidence:\n%s", body)
	}
	assertRouterStatusNoSecrets(t, body)
}

type sequenceChatResult struct {
	result ChatCompletionResult
	err    error
}

type sequenceChatProvider struct {
	results []sequenceChatResult
	calls   int
}

func (s *sequenceChatProvider) ChatCompletion(context.Context, Route, ChatCompletionRequest) (ChatCompletionResult, error) {
	idx := s.calls
	s.calls++
	if idx >= len(s.results) {
		return ChatCompletionResult{}, errors.New("unexpected call")
	}
	return s.results[idx].result, s.results[idx].err
}

func postRouterChat(t *testing.T, handler http.Handler, model, content string) {
	t.Helper()
	postRouterChatWantStatus(t, handler, model, content, http.StatusOK)
}

func postRouterChatWantStatus(t *testing.T, handler http.Handler, model, content string, want int) {
	t.Helper()
	body := strings.NewReader(`{"model":"` + model + `","messages":[{"role":"user","content":"` + content + `"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != want {
		t.Fatalf("chat status = %d, want %d body=%s", rr.Code, want, rr.Body.String())
	}
}

func assertRouterStatusNoSecrets(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{
		"router-local-secret",
		"upstream-secret",
		"wrong-secret",
		"sk-upstream-secret",
		"Authorization: Bearer",
		"?key=",
		"api_key",
		"raw upstream",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("router status/log output leaked %q:\n%s", forbidden, body)
		}
	}
}
