package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRouterStreamingSSEEmitsChunksAndDone(t *testing.T) {
	provider := &scriptedRouterProvider{stream: map[string]scriptedStreamResult{
		"primary-chat": {chunks: []string{"hel", "lo"}},
	}}
	handler := NewHandler(routerFallbackFixtureConfig(), HandlerOptions{LookupEnv: routerServerLookupEnv(), Provider: provider, NowUnix: func() int64 { return 1712345999 }})

	rr := postRouterChatRaw(t, handler, `{"model":"primary-chat","stream":true,"messages":[{"role":"user","content":"hello"}]}`, http.StatusOK)
	body := rr.Body.String()
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	for _, want := range []string{"data: ", `"object":"chat.completion.chunk"`, `"content":"hel"`, `"content":"lo"`, "data: [DONE]"} {
		if !strings.Contains(body, want) {
			t.Fatalf("stream body missing %q:\n%s", want, body)
		}
	}
	if provider.callsFor("primary-chat") != 1 || provider.totalCalls() != 1 {
		t.Fatalf("provider calls = %+v, want one primary stream call", provider.calls)
	}
	assertRouterStatusNoSecrets(t, body)
}

func TestRouterFallbackOnRateLimitBeforeOutput(t *testing.T) {
	provider := &scriptedRouterProvider{chat: map[string][]scriptedChatResult{
		"primary-chat": {{err: ProviderError{Class: "rate_limit", Message: "429 raw body sk-upstream-secret"}}},
		"backup-chat":  {{result: ChatCompletionResult{Content: "fallback answer"}}},
	}}
	handler := NewHandler(routerFallbackFixtureConfig(), HandlerOptions{LookupEnv: routerServerLookupEnv(), Provider: provider})

	rr := postRouterChatRaw(t, handler, `{"model":"primary-chat","messages":[{"role":"user","content":"hello"}]}`, http.StatusOK)
	if !strings.Contains(rr.Body.String(), "fallback answer") {
		t.Fatalf("chat response did not use fallback route:\n%s", rr.Body.String())
	}
	if provider.callsFor("primary-chat") != 1 || provider.callsFor("backup-chat") != 1 {
		t.Fatalf("provider calls = %+v, want primary then backup", provider.calls)
	}
	status := routerStatusBody(t, handler)
	for _, want := range []string{`"attempts":2`, `"failures":1`, `"successes":1`, `"fallbacks":1`, `"last_error_class":"rate_limit"`} {
		if !strings.Contains(status, want) {
			t.Fatalf("status missing %q:\n%s", want, status)
		}
	}
	assertRouterStatusNoSecrets(t, rr.Body.String()+status)
}

func TestRouterNoFallbackAfterStreamStarted(t *testing.T) {
	provider := &scriptedRouterProvider{stream: map[string]scriptedStreamResult{
		"primary-chat": {chunks: []string{"partial"}, streamErr: ProviderError{Class: "server_error", Message: "500 raw body sk-upstream-secret"}},
		"backup-chat":  {chunks: []string{"should-not-run"}},
	}}
	handler := NewHandler(routerFallbackFixtureConfig(), HandlerOptions{LookupEnv: routerServerLookupEnv(), Provider: provider})

	rr := postRouterChatRaw(t, handler, `{"model":"primary-chat","stream":true,"messages":[{"role":"user","content":"hello"}]}`, http.StatusOK)
	body := rr.Body.String()
	for _, want := range []string{`"content":"partial"`, "upstream_stream_interrupted", "data: [DONE]"} {
		if !strings.Contains(body, want) {
			t.Fatalf("stream body missing %q:\n%s", want, body)
		}
	}
	if provider.callsFor("primary-chat") != 1 || provider.callsFor("backup-chat") != 0 || provider.totalCalls() != 1 {
		t.Fatalf("provider calls = %+v, want no fallback after partial output", provider.calls)
	}
	status := routerStatusBody(t, handler)
	if strings.Contains(status, `"fallbacks":1`) {
		t.Fatalf("status recorded fallback after partial stream:\n%s", status)
	}
	assertRouterStatusNoSecrets(t, body+status)
}

func TestRouterNoFallbackOnAuthPolicyOrMalformed(t *testing.T) {
	for _, class := range []string{"auth", "policy", "malformed_request"} {
		t.Run(class, func(t *testing.T) {
			provider := &scriptedRouterProvider{chat: map[string][]scriptedChatResult{
				"primary-chat": {{err: ProviderError{Class: class, Message: "raw body sk-upstream-secret"}}},
				"backup-chat":  {{result: ChatCompletionResult{Content: "should-not-run"}}},
			}}
			handler := NewHandler(routerFallbackFixtureConfig(), HandlerOptions{LookupEnv: routerServerLookupEnv(), Provider: provider})

			rr := postRouterChatRaw(t, handler, `{"model":"primary-chat","messages":[{"role":"user","content":"hello"}]}`, http.StatusBadGateway)
			if provider.callsFor("primary-chat") != 1 || provider.callsFor("backup-chat") != 0 {
				t.Fatalf("provider calls for %s = %+v, want no fallback", class, provider.calls)
			}
			status := routerStatusBody(t, handler)
			if !strings.Contains(status, `"last_error_class":"`+class+`"`) || strings.Contains(status, `"fallbacks":1`) {
				t.Fatalf("status did not record nonfallback class %s safely:\n%s", class, status)
			}
			assertRouterStatusNoSecrets(t, rr.Body.String()+status)
		})
	}
}

type scriptedChatResult struct {
	result ChatCompletionResult
	err    error
}

type scriptedStreamResult struct {
	chunks    []string
	err       error
	streamErr error
}

type scriptedRouterProvider struct {
	chat   map[string][]scriptedChatResult
	stream map[string]scriptedStreamResult
	calls  map[string]int
}

func (s *scriptedRouterProvider) ChatCompletion(_ context.Context, route Route, _ ChatCompletionRequest) (ChatCompletionResult, error) {
	s.recordCall(route.Alias)
	seq := s.chat[route.Alias]
	idx := s.calls[route.Alias] - 1
	if idx < 0 || idx >= len(seq) {
		return ChatCompletionResult{}, errors.New("unexpected chat call")
	}
	return seq[idx].result, seq[idx].err
}

func (s *scriptedRouterProvider) StreamChatCompletion(_ context.Context, route Route, _ ChatCompletionRequest) (ChatStreamResult, error) {
	s.recordCall(route.Alias)
	result, ok := s.stream[route.Alias]
	if !ok {
		return ChatStreamResult{}, errors.New("unexpected stream call")
	}
	return ChatStreamResult{Chunks: result.chunks, Err: result.streamErr}, result.err
}

func (s *scriptedRouterProvider) recordCall(alias string) {
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[alias]++
}

func (s *scriptedRouterProvider) callsFor(alias string) int {
	if s.calls == nil {
		return 0
	}
	return s.calls[alias]
}

func (s *scriptedRouterProvider) totalCalls() int {
	total := 0
	for _, count := range s.calls {
		total += count
	}
	return total
}

func routerFallbackFixtureConfig() config.Config {
	cfg := routerServerFixtureConfig()
	cfg.Router.Routes = []config.RouterRouteCfg{
		{Name: "primary-provider", Alias: "primary-chat", Provider: "custom", Model: "fake-model", BaseURL: "https://llm.example/v1", APIKeyEnv: "UPSTREAM_KEY", Transport: DefaultTransport},
		{Name: "backup-provider", Alias: "backup-chat", Provider: "custom", Model: "backup-model", BaseURL: "https://backup.example/v1", APIKeyEnv: "UPSTREAM_KEY", Transport: DefaultTransport},
	}
	cfg.Router.Fallback = []config.RouterFallbackCfg{{From: "primary-chat", To: "backup-chat", On: []string{"rate_limit", "server_error", "timeout", "connection_failure"}}}
	return cfg
}

func postRouterChatRaw(t *testing.T, handler http.Handler, body string, want int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != want {
		t.Fatalf("chat status = %d, want %d body=%s", rr.Code, want, rr.Body.String())
	}
	return rr
}

func routerStatusBody(t *testing.T, handler http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}
