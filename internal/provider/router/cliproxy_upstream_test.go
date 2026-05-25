package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRouterCLIProxyAPIUpstream(t *testing.T) {
	const upstreamSecret = "cliproxy-upstream-secret"
	var seenPaths []string
	var chatBody struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer "+upstreamSecret {
			t.Fatalf("upstream Authorization = %q, want bearer from api_key_env", got)
		}
		switch r.URL.Path {
		case "/v1/models":
			if r.Method != http.MethodGet {
				t.Fatalf("models method = %s, want GET", r.Method)
			}
			writeRouterJSON(w, http.StatusOK, map[string]any{
				"object": "list",
				"data":   []map[string]any{{"id": "gpt-cliproxy", "object": "model"}},
			})
		case "/v1/chat/completions":
			if r.Method != http.MethodPost {
				t.Fatalf("chat method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&chatBody); err != nil {
				t.Fatalf("decode chat body: %v", err)
			}
			writeRouterJSON(w, http.StatusOK, map[string]any{
				"id":      "chatcmpl-cliproxy-fixture",
				"object":  "chat.completion",
				"created": int64(1712346000),
				"model":   "gpt-cliproxy",
				"choices": []map[string]any{{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "hello through CLIProxyAPI-compatible upstream",
					},
					"finish_reason": "stop",
				}},
				"usage": map[string]int{"prompt_tokens": 4, "completion_tokens": 6, "total_tokens": 10},
			})
		default:
			t.Fatalf("unexpected upstream path %s; adapter must use only /v1/models and /v1/chat/completions", r.URL.Path)
		}
	}))
	defer upstream.Close()

	lookup := mapLookup(map[string]string{
		"GORMES_ROUTER_API_KEY": "router-local-secret",
		"CLIPROXY_API_KEY":      upstreamSecret,
	})
	cfg := config.Config{Router: config.RouterCfg{
		Enabled:   true,
		Listen:    "127.0.0.1:8787",
		APIKeyEnv: "GORMES_ROUTER_API_KEY",
		Routes: []config.RouterRouteCfg{{
			Name:      "cliproxy-compatible",
			Alias:     "cliproxy-chat",
			Provider:  "custom",
			Model:     "gpt-cliproxy",
			BaseURL:   upstream.URL + "/v1",
			APIKeyEnv: "CLIPROXY_API_KEY",
			Transport: DefaultTransport,
		}},
	}}

	model := BuildReadModel(cfg, Options{LookupEnv: lookup})
	if got, want := model.Status.State, RouterStatusConfigured; got != want {
		t.Fatalf("router status = %q, want %q (model=%+v)", got, want, model)
	}
	if len(model.Routes) != 1 || model.Routes[0].APIKeyEnv != "CLIPROXY_API_KEY" {
		t.Fatalf("route credential env = %+v, want redacted api_key_env handle for adapter", model.Routes)
	}

	provider := NewHTTPUpstreamProvider(HTTPUpstreamProviderOptions{LookupEnv: lookup, Client: upstream.Client()})
	probe := provider.Probe(context.Background(), model.Routes[0])
	if !probe.Available || !contains(probe.Evidence, "models_probe_available") {
		t.Fatalf("probe = %+v, want available models evidence", probe)
	}

	handler := NewHandler(cfg, HandlerOptions{
		LookupEnv: lookup,
		NowUnix:   func() int64 { return 1712346001 },
	})
	reqBody := []byte(`{"model":"cliproxy-chat","messages":[{"role":"user","content":"ping"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer router-local-secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("chat status = %d body=%s", rr.Code, rr.Body.String())
	}
	if chatBody.Model != "gpt-cliproxy" || chatBody.Stream {
		t.Fatalf("upstream chat body model=%q stream=%t, want configured upstream model non-streaming", chatBody.Model, chatBody.Stream)
	}
	if len(chatBody.Messages) != 1 || chatBody.Messages[0].Content != "ping" {
		t.Fatalf("upstream messages = %+v", chatBody.Messages)
	}
	if !strings.Contains(rr.Body.String(), "hello through CLIProxyAPI-compatible upstream") {
		t.Fatalf("router response missing upstream content:\n%s", rr.Body.String())
	}
	assertRouterStatusNoSecrets(t, rr.Body.String())
	if strings.Contains(rr.Body.String(), upstreamSecret) || strings.Contains(rr.Body.String(), "CLIPROXY_API_KEY") {
		t.Fatalf("router response leaked upstream credential evidence:\n%s", rr.Body.String())
	}
	if want := []string{"/v1/models", "/v1/chat/completions"}; !reflect.DeepEqual(seenPaths, want) {
		t.Fatalf("upstream paths = %#v, want %#v", seenPaths, want)
	}
}

func TestRouterCLIProxyAPIUpstreamRedactsHTTPFailures(t *testing.T) {
	const upstreamSecret = "cliproxy-upstream-secret"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+upstreamSecret {
			t.Fatalf("upstream Authorization = %q, want bearer from api_key_env", got)
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`raw upstream body with sk-upstream-secret and Authorization: Bearer cliproxy-upstream-secret`))
	}))
	defer upstream.Close()

	lookup := mapLookup(map[string]string{
		"GORMES_ROUTER_API_KEY": "router-local-secret",
		"CLIPROXY_API_KEY":      upstreamSecret,
	})
	cfg := config.Config{Router: config.RouterCfg{
		Enabled:   true,
		Listen:    "127.0.0.1:8787",
		APIKeyEnv: "GORMES_ROUTER_API_KEY",
		Routes: []config.RouterRouteCfg{{
			Name:      "cliproxy-compatible",
			Alias:     "cliproxy-chat",
			Provider:  "custom",
			Model:     "gpt-cliproxy",
			BaseURL:   upstream.URL + "/v1",
			APIKeyEnv: "CLIPROXY_API_KEY",
			Transport: DefaultTransport,
		}},
	}}
	handler := NewHandler(cfg, HandlerOptions{LookupEnv: lookup})
	rr := postRouterChatRaw(t, handler, `{"model":"cliproxy-chat","messages":[{"role":"user","content":"ping"}]}`, http.StatusBadGateway)
	body := rr.Body.String()
	if !strings.Contains(body, "rate_limit") {
		t.Fatalf("body = %s, want safe rate_limit code", body)
	}
	assertRouterStatusNoSecrets(t, body+routerStatusBody(t, handler))
	for _, forbidden := range []string{upstreamSecret, "sk-upstream-secret", "CLIPROXY_API_KEY", "raw upstream body"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("router response leaked %q:\n%s", forbidden, body)
		}
	}
}
