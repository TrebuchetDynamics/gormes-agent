package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRouterModelsListsConfiguredAliases(t *testing.T) {
	handler := NewHandler(routerServerFixtureConfig(), HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  fakeChatProvider{reply: "unused"},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var got struct {
		Object string `json:"object"`
		Data   []struct {
			ID     string `json:"id"`
			Object string `json:"object"`
			Owned  string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("models JSON: %v\n%s", err, rr.Body.String())
	}
	if got.Object != "list" {
		t.Fatalf("object = %q, want list", got.Object)
	}
	if len(got.Data) != 1 || got.Data[0].ID != "primary-chat" || got.Data[0].Object != "model" || got.Data[0].Owned != "custom" {
		t.Fatalf("models = %+v, want configured primary-chat only", got.Data)
	}
	if strings.Contains(rr.Body.String(), "missing-chat") || strings.Contains(rr.Body.String(), "upstream-secret") || strings.Contains(rr.Body.String(), "router-local-secret") {
		t.Fatalf("models response leaked excluded alias or secret:\n%s", rr.Body.String())
	}
}

func TestRouterAuthAcceptsBearerAndXAPIKeyAndRejectsMissingOrBad(t *testing.T) {
	handler := NewHandler(routerServerFixtureConfig(), HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  fakeChatProvider{reply: "unused"},
	})

	for _, tc := range []struct {
		name   string
		header string
		value  string
	}{
		{name: "bearer", header: "Authorization", value: "Bearer router-local-secret"},
		{name: "x-api-key", header: "X-Api-Key", value: "router-local-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			req.Header.Set(tc.header, tc.value)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
			}
		})
	}

	for _, tc := range []struct {
		name   string
		header string
		value  string
	}{
		{name: "missing"},
		{name: "bad-bearer", header: "Authorization", value: "Bearer wrong-secret"},
		{name: "bad-x-api-key", header: "X-Api-Key", value: "wrong-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
			}
			body := rr.Body.String()
			if !strings.Contains(body, "router_unauthorized") || strings.Contains(body, "router-local-secret") || strings.Contains(body, "wrong-secret") {
				t.Fatalf("401 body missing safe error or leaked secret:\n%s", body)
			}
		})
	}
}

func TestRouterChatCompletionsNonStreaming(t *testing.T) {
	provider := &recordingChatProvider{reply: "hello from fake upstream"}
	handler := NewHandler(routerServerFixtureConfig(), HandlerOptions{
		LookupEnv: routerServerLookupEnv(),
		Provider:  provider,
		NowUnix:   func() int64 { return 1712345678 },
	})

	body := []byte(`{"model":"primary-chat","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider requests = %d, want 1", len(provider.requests))
	}
	if provider.requests[0].Route.Alias != "primary-chat" || provider.requests[0].Request.Model != "primary-chat" || provider.requests[0].Request.Messages[0].Content != "hello" {
		t.Fatalf("provider request = %+v", provider.requests[0])
	}
	var got struct {
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("chat JSON: %v\n%s", err, rr.Body.String())
	}
	if got.Object != "chat.completion" || got.Created != 1712345678 || got.Model != "primary-chat" || len(got.Choices) != 1 || got.Choices[0].Message.Role != "assistant" || got.Choices[0].Message.Content != "hello from fake upstream" || got.Choices[0].FinishReason != "stop" {
		t.Fatalf("chat response = %+v", got)
	}
	if strings.Contains(rr.Body.String(), "upstream-secret") || strings.Contains(rr.Body.String(), "router-local-secret") {
		t.Fatalf("chat response leaked secret:\n%s", rr.Body.String())
	}
}

type fakeChatProvider struct{ reply string }

func (f fakeChatProvider) ChatCompletion(context.Context, Route, ChatCompletionRequest) (ChatCompletionResult, error) {
	return ChatCompletionResult{Content: f.reply}, nil
}

type recordingChatProvider struct {
	reply    string
	requests []recordedRouterRequest
}

type recordedRouterRequest struct {
	Route   Route
	Request ChatCompletionRequest
}

func (f *recordingChatProvider) ChatCompletion(_ context.Context, route Route, req ChatCompletionRequest) (ChatCompletionResult, error) {
	f.requests = append(f.requests, recordedRouterRequest{Route: route, Request: req})
	return ChatCompletionResult{Content: f.reply}, nil
}

func routerServerFixtureConfig() config.Config {
	return config.Config{Router: config.RouterCfg{
		Enabled:   true,
		Listen:    "127.0.0.1:8787",
		APIKeys:   []string{"router-local-secret"},
		APIKeyEnv: "GORMES_ROUTER_API_KEY",
		Routes: []config.RouterRouteCfg{
			{Name: "primary-provider", Alias: "primary-chat", Provider: "custom", Model: "fake-model", BaseURL: "https://llm.example/v1", APIKeyEnv: "UPSTREAM_KEY", Transport: DefaultTransport},
			{Name: "missing-provider", Alias: "missing-chat", Provider: "custom", Model: "missing-model", BaseURL: "https://missing.example/v1", APIKeyEnv: "MISSING_UPSTREAM_KEY", Transport: DefaultTransport},
		},
	}}
}

func routerServerLookupEnv() func(string) (string, bool) {
	return mapLookup(map[string]string{
		"GORMES_ROUTER_API_KEY": "router-local-secret",
		"UPSTREAM_KEY":          "upstream-secret",
	})
}
