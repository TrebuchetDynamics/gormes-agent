package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebToolsExposeHermesNamesAndSchemas(t *testing.T) {
	webTools := NewWebTools(WebToolsConfig{
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			Available: true,
		},
	})
	if len(webTools) != 3 {
		t.Fatalf("NewWebTools len = %d, want 3", len(webTools))
	}
	byName := map[string]Tool{}
	for _, tool := range webTools {
		byName[tool.Name()] = tool
		if strings.TrimSpace(tool.Description()) == "" {
			t.Fatalf("%s description is empty", tool.Name())
		}
		if !json.Valid(tool.Schema()) {
			t.Fatalf("%s schema is invalid JSON: %s", tool.Name(), tool.Schema())
		}
	}
	for _, name := range []string{WebToolSearch, WebToolExtract, WebToolCrawl} {
		if byName[name] == nil {
			t.Fatalf("%s not exposed by NewWebTools", name)
		}
	}
	if !strings.Contains(string(byName[WebToolSearch].Schema()), `"query"`) {
		t.Fatalf("web_search schema missing query: %s", byName[WebToolSearch].Schema())
	}
	if !strings.Contains(string(byName[WebToolSearch].Schema()), `"limit"`) {
		t.Fatalf("web_search schema missing Hermes limit parameter: %s", byName[WebToolSearch].Schema())
	}
	if !strings.Contains(string(byName[WebToolExtract].Schema()), `"urls"`) {
		t.Fatalf("web_extract schema missing urls: %s", byName[WebToolExtract].Schema())
	}
	if !strings.Contains(string(byName[WebToolCrawl].Schema()), `"url"`) ||
		!strings.Contains(string(byName[WebToolCrawl].Schema()), `"instructions"`) ||
		!strings.Contains(string(byName[WebToolCrawl].Schema()), `"depth"`) {
		t.Fatalf("web_crawl schema missing Hermes crawl parameters: %s", byName[WebToolCrawl].Schema())
	}
}

func TestResolveWebBackendPrefersFirecrawlConfig(t *testing.T) {
	resolved := ResolveWebBackend(map[string]string{
		"FIRECRAWL_API_URL": " https://firecrawl.example/v2/ ",
		"FIRECRAWL_API_KEY": "fire-key",
	})
	if !resolved.Available {
		t.Fatalf("resolved.Available = false, want true: %+v", resolved)
	}
	if resolved.Backend != WebBackendFirecrawl {
		t.Fatalf("Backend = %q, want %q", resolved.Backend, WebBackendFirecrawl)
	}
	if resolved.BaseURL != "https://firecrawl.example" {
		t.Fatalf("BaseURL = %q, want trimmed origin", resolved.BaseURL)
	}
	if resolved.APIKey != "fire-key" {
		t.Fatalf("APIKey = %q, want fire-key", resolved.APIKey)
	}

	fallback := ResolveWebBackend(map[string]string{})
	if !fallback.Available || fallback.Backend != WebBackendDuckDuckGo || fallback.Source != "free" {
		t.Fatalf("empty env resolved as %+v, want automatic DuckDuckGo fallback", fallback)
	}
	if fallback.BaseURL != webDefaultDuckDuckGoBaseURL {
		t.Fatalf("fallback BaseURL = %q, want %q", fallback.BaseURL, webDefaultDuckDuckGoBaseURL)
	}
	status := ResolveWebBackendStatus(map[string]string{}, WebBackendConfig{})
	if !status.Available || status.Backend != WebBackendDuckDuckGo || status.Route != "direct" {
		t.Fatalf("status = %+v, want automatic DuckDuckGo direct route", status)
	}
	if strings.Join(status.ToolNames, ",") != strings.Join([]string{WebToolSearch, WebToolExtract}, ",") {
		t.Fatalf("ToolNames = %v, want DuckDuckGo search plus Instant Answer extract", status.ToolNames)
	}
}

func TestResolveWebBackendHonorsHermesBackendConfig(t *testing.T) {
	env := map[string]string{
		"FIRECRAWL_API_KEY": "fire-key",
		"PARALLEL_API_KEY":  "parallel-key",
		"TAVILY_API_KEY":    "tavily-key",
		"EXA_API_KEY":       "exa-key",
	}
	for _, tc := range []struct {
		name    string
		backend string
		want    WebBackend
		apiKey  string
	}{
		{name: "parallel", backend: " parallel ", want: WebBackendParallel, apiKey: "parallel-key"},
		{name: "exa", backend: "EXA", want: WebBackendExa, apiKey: "exa-key"},
		{name: "tavily", backend: "tavily", want: WebBackendTavily, apiKey: "tavily-key"},
		{name: "firecrawl", backend: "firecrawl", want: WebBackendFirecrawl, apiKey: "fire-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolved := ResolveWebBackendWithConfig(env, WebBackendConfig{Backend: tc.backend})
			if !resolved.Available {
				t.Fatalf("resolved.Available = false, want true: %+v", resolved)
			}
			if resolved.Backend != tc.want {
				t.Fatalf("Backend = %q, want %q", resolved.Backend, tc.want)
			}
			if resolved.APIKey != tc.apiKey {
				t.Fatalf("APIKey = %q, want configured backend key", resolved.APIKey)
			}
		})
	}

	fallback := ResolveWebBackendWithConfig(map[string]string{
		"FIRECRAWL_API_KEY": "   ",
		"PARALLEL_API_KEY":  "parallel-key",
		"TAVILY_API_KEY":    "tavily-key",
		"EXA_API_KEY":       "exa-key",
	}, WebBackendConfig{})
	if fallback.Backend != WebBackendParallel || fallback.APIKey != "parallel-key" || !fallback.Available {
		t.Fatalf("fallback = %+v, want parallel after blank Firecrawl key is ignored", fallback)
	}
}

func TestResolveWebBackendSupportsCDPExtractFallback(t *testing.T) {
	env := map[string]string{
		"CHROME_REMOTE_DEBUGGING_URL": " http://127.0.0.1:9222 ",
	}
	resolved := ResolveWebBackendWithConfig(env, WebBackendConfig{Backend: " browser "})
	if !resolved.Available {
		t.Fatalf("resolved.Available = false, want local CDP backend: %+v", resolved)
	}
	if resolved.Backend != WebBackendCDP {
		t.Fatalf("Backend = %q, want %q", resolved.Backend, WebBackendCDP)
	}
	if resolved.BaseURL != "http://127.0.0.1:9222" || resolved.Source != "env" {
		t.Fatalf("resolved = %+v, want trimmed CHROME_REMOTE_DEBUGGING_URL env source", resolved)
	}

	auto := ResolveWebBackendWithConfig(env, WebBackendConfig{})
	if auto.Backend != WebBackendDuckDuckGo || !auto.Available {
		t.Fatalf("auto fallback = %+v, want DuckDuckGo search route with CDP reserved for extract fallback", auto)
	}

	status := ResolveWebBackendStatus(env, WebBackendConfig{})
	if !status.Available || status.Route != "direct" || status.Backend != WebBackendDuckDuckGo {
		t.Fatalf("status = %+v, want available DuckDuckGo direct route", status)
	}
	if strings.Join(status.ToolNames, ",") != strings.Join([]string{WebToolSearch, WebToolExtract}, ",") {
		t.Fatalf("ToolNames = %v, want search plus extract with CDP fallback metadata", status.ToolNames)
	}
	if !stringSliceContains(status.RequiresEnv, "CHROME_REMOTE_DEBUGGING_URL") {
		t.Fatalf("RequiresEnv = %v, want CDP env metadata", status.RequiresEnv)
	}
	if !stringSliceContains(status.RequiresEnv, "BROWSER_CDP_URL") {
		t.Fatalf("RequiresEnv = %v, want Hermes BROWSER_CDP_URL metadata", status.RequiresEnv)
	}

	browserAlias := ResolveWebBackendWithConfig(map[string]string{
		"BROWSER_CDP_URL": " http://127.0.0.1:9223 ",
	}, WebBackendConfig{Backend: "cdp"})
	if !browserAlias.Available || browserAlias.Backend != WebBackendCDP || browserAlias.BaseURL != "http://127.0.0.1:9223" {
		t.Fatalf("browserAlias = %+v, want explicit CDP backend from BROWSER_CDP_URL", browserAlias)
	}
}

func TestResolveWebBackendSupportsAdditionalSearchBackends(t *testing.T) {
	brave := ResolveWebBackendWithConfig(map[string]string{
		"BRAVE_API_KEY": "brave-secret",
	}, WebBackendConfig{Backend: "brave_search"})
	if !brave.Available || brave.Backend != WebBackendBrave || brave.APIKey != "brave-secret" || brave.BaseURL != webDefaultBraveBaseURL {
		t.Fatalf("brave = %+v, want configured Brave Search backend", brave)
	}

	searxng := ResolveWebBackendWithConfig(map[string]string{
		"SEARXNG_BASE_URL": " https://search.example.test/ ",
	}, WebBackendConfig{Backend: "searx"})
	if !searxng.Available || searxng.Backend != WebBackendSearXNG || searxng.BaseURL != "https://search.example.test" {
		t.Fatalf("searxng = %+v, want configured SearXNG backend", searxng)
	}

	duck := ResolveWebBackendWithConfig(map[string]string{}, WebBackendConfig{Backend: "ddg"})
	if !duck.Available || duck.Backend != WebBackendDuckDuckGo || duck.BaseURL != webDefaultDuckDuckGoBaseURL {
		t.Fatalf("duck = %+v, want explicit DuckDuckGo backend", duck)
	}

	perplexity := ResolveWebBackendWithConfig(map[string]string{
		"PERPLEXITY_API_KEY": "perplexity-secret",
	}, WebBackendConfig{Backend: "perplexity"})
	if !perplexity.Available || perplexity.Backend != WebBackendPerplexity || perplexity.APIKey != "perplexity-secret" || perplexity.BaseURL != webDefaultPerplexityBaseURL {
		t.Fatalf("perplexity = %+v, want configured Perplexity backend", perplexity)
	}

	auto := ResolveWebBackendWithConfig(map[string]string{
		"BRAVE_API_KEY":               "brave-secret",
		"SEARXNG_BASE_URL":            "https://search.example.test",
		"PERPLEXITY_API_KEY":          "perplexity-secret",
		"DUCKDUCKGO_ENABLED":          "1",
		"TAVILY_API_KEY":              "tavily-secret",
		"FIRECRAWL_API_KEY":           " ",
		"PARALLEL_API_KEY":            " ",
		"CHROME_REMOTE_DEBUGGING_URL": "http://127.0.0.1:9222",
	}, WebBackendConfig{})
	if auto.Backend != WebBackendTavily {
		t.Fatalf("auto = %+v, want existing Hermes backend order to beat new search-only backends", auto)
	}

	autoBrave := ResolveWebBackendWithConfig(map[string]string{
		"BRAVE_API_KEY":    "brave-secret",
		"SEARXNG_BASE_URL": "https://search.example.test",
	}, WebBackendConfig{})
	if autoBrave.Backend != WebBackendBrave || !autoBrave.Available {
		t.Fatalf("autoBrave = %+v, want Brave before SearXNG when hosted Hermes providers are absent", autoBrave)
	}

	autoPerplexity := ResolveWebBackendWithConfig(map[string]string{
		"PERPLEXITY_API_KEY": "perplexity-secret",
		"DUCKDUCKGO_ENABLED": "1",
	}, WebBackendConfig{})
	if autoPerplexity.Backend != WebBackendPerplexity || !autoPerplexity.Available {
		t.Fatalf("autoPerplexity = %+v, want Perplexity before DuckDuckGo when configured", autoPerplexity)
	}

	status := ResolveWebBackendStatus(map[string]string{"BRAVE_API_KEY": "brave-secret"}, WebBackendConfig{})
	if len(status.ToolNames) != 1 || status.ToolNames[0] != WebToolSearch {
		t.Fatalf("ToolNames = %v, want search-only backend to advertise web_search", status.ToolNames)
	}
	if stringSliceContains(status.RequiresEnv, "DUCKDUCKGO_ENABLED") {
		t.Fatalf("RequiresEnv = %v, must not require obsolete DuckDuckGo toggle", status.RequiresEnv)
	}
}

func TestResolveWebBackendUsesNousAuthStoreForManagedGateway(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{
  "providers": {
    "nous": {
      "access_token": "nous-access-token",
      "expires_at": "2999-01-01T00:00:00Z"
    }
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	resolved := ResolveWebBackendWithConfig(map[string]string{
		"FIRECRAWL_API_KEY": "direct-firecrawl-key",
	}, WebBackendConfig{
		Backend:             "firecrawl",
		UseGateway:          true,
		ManagedToolsEnabled: true,
		AuthStorePath:       authPath,
	})
	if !resolved.Available {
		t.Fatalf("resolved.Available = false, want managed gateway: %+v", resolved)
	}
	if resolved.Backend != WebBackendFirecrawl {
		t.Fatalf("Backend = %q, want firecrawl", resolved.Backend)
	}
	if resolved.BaseURL != "https://firecrawl-gateway.nousresearch.com" {
		t.Fatalf("BaseURL = %q, want default managed Firecrawl gateway", resolved.BaseURL)
	}
	if resolved.APIKey != "nous-access-token" {
		t.Fatalf("APIKey = %q, want auth-store access token", resolved.APIKey)
	}
	if !resolved.Managed || resolved.Source != "auth_store" {
		t.Fatalf("managed source = (%t, %q), want auth_store managed route", resolved.Managed, resolved.Source)
	}
}

func TestResolveWebBackendStatusReportsManagedRouteAndToolset(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{
  "providers": {
    "nous": {
      "access_token": "nous-status-token",
      "expires_at": "2999-01-01T00:00:00Z"
    }
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	status := ResolveWebBackendStatus(map[string]string{
		"FIRECRAWL_GATEWAY_URL": "https://firecrawl-gateway.test",
	}, WebBackendConfig{
		Backend:             "firecrawl",
		UseGateway:          true,
		ManagedToolsEnabled: true,
		AuthStorePath:       authPath,
	})
	if !status.Available || status.Route != "managed" || status.Source != "auth_store" {
		t.Fatalf("status = %+v, want managed auth-store route", status)
	}
	if status.Backend != WebBackendFirecrawl || status.BaseURL != "https://firecrawl-gateway.test" {
		t.Fatalf("status backend/base = %q/%q, want configured Firecrawl gateway", status.Backend, status.BaseURL)
	}
	for _, name := range []string{WebToolSearch, WebToolExtract, WebToolCrawl} {
		if !stringSliceContains(status.ToolNames, name) {
			t.Fatalf("ToolNames = %v, missing %s", status.ToolNames, name)
		}
	}
	if !stringSliceContains(status.RequiresEnv, "TOOL_GATEWAY_USER_TOKEN") {
		t.Fatalf("RequiresEnv = %v, want managed gateway env metadata", status.RequiresEnv)
	}
	if strings.Contains(fmt.Sprintf("%+v", status), "nous-status-token") {
		t.Fatalf("status leaked token: %+v", status)
	}
}

func TestResolveWebBackendRefreshesExpiringNousAuthStoreToken(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	var seenForm string
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oauth/token" {
			t.Errorf("refresh path = %q, want /api/oauth/token", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("refresh method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		seenForm = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fresh-nous-token","refresh_token":"rotated-refresh-token","token_type":"Bearer","expires_in":3600,"scope":"inference:mint_agent_key"}`))
	}))
	defer portal.Close()

	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(fmt.Sprintf(`{
  "providers": {
    "nous": {
      "access_token": "stale-nous-token",
      "refresh_token": "old-refresh-token",
      "client_id": "hermes-cli",
      "portal_base_url": %q,
      "expires_at": "2026-04-30T11:59:00Z"
    }
  },
  "credential_pool": {
    "nous": []
  }
}`, portal.URL)), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	resolved := ResolveWebBackendWithConfig(map[string]string{}, WebBackendConfig{
		Backend:             "firecrawl",
		UseGateway:          true,
		ManagedToolsEnabled: true,
		AuthStorePath:       authPath,
		Now:                 func() time.Time { return now },
	})
	if !resolved.Available || resolved.APIKey != "fresh-nous-token" || resolved.Source != "auth_store" || !resolved.Managed {
		t.Fatalf("resolved = %+v, want refreshed auth-store managed token", resolved)
	}
	if !strings.Contains(seenForm, "grant_type=refresh_token") || !strings.Contains(seenForm, "refresh_token=old-refresh-token") {
		t.Fatalf("refresh form = %q, want refresh token grant", seenForm)
	}

	raw, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("read refreshed auth store: %v", err)
	}
	var store map[string]any
	if err := json.Unmarshal(raw, &store); err != nil {
		t.Fatalf("decode refreshed auth store: %v", err)
	}
	providers, _ := store["providers"].(map[string]any)
	nous, _ := providers["nous"].(map[string]any)
	if nous["access_token"] != "fresh-nous-token" || nous["refresh_token"] != "rotated-refresh-token" {
		t.Fatalf("providers.nous = %#v, want rotated tokens persisted", nous)
	}
	if _, ok := store["credential_pool"]; !ok {
		t.Fatalf("refreshed store dropped credential_pool: %#v", store)
	}
}

func TestResolveWebBackendFallsBackToCachedNousTokenWhenRefreshFails(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"temporarily_unavailable"}`, http.StatusBadGateway)
	}))
	defer portal.Close()

	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(fmt.Sprintf(`{
  "providers": {
    "nous": {
      "access_token": "cached-nous-token",
      "refresh_token": "refresh-token",
      "portal_base_url": %q,
      "expires_at": "2026-04-30T11:59:00Z"
    }
  }
}`, portal.URL)), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	resolved := ResolveWebBackendWithConfig(nil, WebBackendConfig{
		Backend:             "firecrawl",
		UseGateway:          true,
		ManagedToolsEnabled: true,
		AuthStorePath:       authPath,
		Now:                 func() time.Time { return now },
	})
	if !resolved.Available || resolved.APIKey != "cached-nous-token" {
		t.Fatalf("resolved = %+v, want cached token fallback when refresh fails", resolved)
	}
}

func TestResolveWebBackendRejectsInvalidManagedGatewayScheme(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{
  "providers": {
    "nous": {
      "access_token": "nous-token",
      "expires_at": "2999-01-01T00:00:00Z"
    }
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	resolved := ResolveWebBackendWithConfig(map[string]string{
		"TOOL_GATEWAY_DOMAIN": "nousresearch.com",
		"TOOL_GATEWAY_SCHEME": "ftp",
	}, WebBackendConfig{
		Backend:             "firecrawl",
		UseGateway:          true,
		ManagedToolsEnabled: true,
		AuthStorePath:       authPath,
	})
	if resolved.Available || resolved.Source != "invalid_gateway_scheme" {
		t.Fatalf("resolved = %+v, want unavailable invalid gateway scheme", resolved)
	}
}

func TestResolveWebBackendInvalidGatewaySchemeDoesNotBlockDirectFirecrawl(t *testing.T) {
	resolved := ResolveWebBackendWithConfig(map[string]string{
		"FIRECRAWL_API_KEY":       "direct-firecrawl-key",
		"TOOL_GATEWAY_DOMAIN":     "nousresearch.com",
		"TOOL_GATEWAY_SCHEME":     "ftp",
		"TOOL_GATEWAY_USER_TOKEN": "nous-token",
	}, WebBackendConfig{Backend: "firecrawl"})
	if !resolved.Available || resolved.APIKey != "direct-firecrawl-key" || resolved.Source != "env" || resolved.Managed {
		t.Fatalf("resolved = %+v, want direct Firecrawl to win when gateway mode is not requested", resolved)
	}
}

func TestWebSearchToolCallsFirecrawlThroughWebHTTPClient(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":[{"title":"Gormes","url":"https://example.test/gormes","description":"Go agent"}]}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes agent","limit":3}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/v2/search" {
		t.Fatalf("request = %s %s, want POST /v2/search", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer fire-secret" {
		t.Fatalf("Authorization = %q, want bearer token", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["query"] != "gormes agent" || sent["limit"] != float64(3) {
		t.Fatalf("search body = %#v, want query + limit", sent)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Web []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
				Position    int    `json:"position"`
			} `json:"web"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 1 {
		t.Fatalf("output = %+v, want one successful web result", payload)
	}
	if payload.Data.Web[0].Position != 1 {
		t.Fatalf("position = %d, want 1", payload.Data.Web[0].Position)
	}
}

func TestWebSearchToolCallsTavilyBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Tavily Result","url":"https://example.test/tavily","content":"Tavily snippet"}]}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendTavily,
			BaseURL:   "https://tavily.test",
			APIKey:    "tavily-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes tavily","limit":40}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/search" {
		t.Fatalf("request = %s %s, want POST /search", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want Tavily key in JSON body for Hermes parity", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["query"] != "gormes tavily" || sent["api_key"] != "tavily-secret" {
		t.Fatalf("tavily body = %#v, want query + api_key", sent)
	}
	if sent["max_results"] != float64(20) {
		t.Fatalf("max_results = %v, want Hermes Tavily cap of 20", sent["max_results"])
	}
	if sent["include_raw_content"] != false || sent["include_images"] != false {
		t.Fatalf("tavily body = %#v, want raw content and images disabled", sent)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Web []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
				Position    int    `json:"position"`
			} `json:"web"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 1 {
		t.Fatalf("output = %+v, want one successful Tavily result", payload)
	}
	if payload.Data.Web[0].Description != "Tavily snippet" || payload.Data.Web[0].Position != 1 {
		t.Fatalf("normalized result = %+v, want Tavily content as description", payload.Data.Web[0])
	}
}

func TestWebSearchToolCallsExaBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Exa Result","url":"https://example.test/exa","highlights":["first excerpt","second excerpt"]}]}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendExa,
			BaseURL:   "https://exa.test",
			APIKey:    "exa-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes exa","limit":4}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Exa search call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/search" {
		t.Fatalf("request = %s %s, want POST /search", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("x-api-key"); got != "exa-secret" {
		t.Fatalf("x-api-key = %q, want Exa key", got)
	}
	if got := req.Header.Get("x-exa-integration"); got != "hermes-agent" {
		t.Fatalf("x-exa-integration = %q, want hermes-agent", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["query"] != "gormes exa" || sent["numResults"] != float64(4) {
		t.Fatalf("exa body = %#v, want query + numResults", sent)
	}
	contents, _ := sent["contents"].(map[string]any)
	if contents["highlights"] != true {
		t.Fatalf("contents = %#v, want highlights enabled", contents)
	}

	var payload struct {
		Data struct {
			Web []struct {
				Description string `json:"description"`
			} `json:"web"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if got := payload.Data.Web[0].Description; got != "first excerpt second excerpt" {
		t.Fatalf("description = %q, want joined highlights", got)
	}
}

func TestWebSearchToolCallsParallelBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Parallel Result","url":"https://example.test/parallel","excerpts":["parallel one","parallel two"]}]}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendParallel,
			BaseURL:   "https://parallel.test",
			APIKey:    "parallel-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes parallel","limit":40}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Parallel search call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/v1beta/search" {
		t.Fatalf("request = %s %s, want POST /v1beta/search", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("x-api-key"); got != "parallel-secret" {
		t.Fatalf("x-api-key = %q, want Parallel key", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["objective"] != "gormes parallel" || sent["mode"] != "agentic" {
		t.Fatalf("parallel body = %#v, want objective + default mode", sent)
	}
	if sent["max_results"] != float64(20) {
		t.Fatalf("max_results = %v, want Hermes Parallel cap of 20", sent["max_results"])
	}

	var payload struct {
		Data struct {
			Web []struct {
				Description string `json:"description"`
			} `json:"web"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if got := payload.Data.Web[0].Description; got != "parallel one parallel two" {
		t.Fatalf("description = %q, want joined excerpts", got)
	}
}

func TestWebSearchToolCallsBraveBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"web":{"results":[{"title":"Brave Result","url":"https://example.test/brave","description":"Brave snippet"}]}}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendBrave,
			BaseURL:   "https://api.search.brave.test",
			APIKey:    "brave-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes brave","limit":40}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Brave search call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/res/v1/web/search" {
		t.Fatalf("request = %s %s, want GET /res/v1/web/search", req.Method, req.URL.Path)
	}
	if req.URL.Query().Get("q") != "gormes brave" || req.URL.Query().Get("count") != "20" {
		t.Fatalf("query = %s, want q + capped count", req.URL.RawQuery)
	}
	if got := req.Header.Get("X-Subscription-Token"); got != "brave-secret" {
		t.Fatalf("X-Subscription-Token = %q, want Brave API key", got)
	}

	var payload webSearchResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 1 || payload.Data.Web[0].Description != "Brave snippet" {
		t.Fatalf("output = %+v, want normalized Brave result", payload)
	}
}

func TestWebSearchToolCallsSearXNGBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"SearXNG Result","url":"https://example.test/searx","content":"SearXNG snippet"}]}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendSearXNG,
			BaseURL:   "https://search.example.test",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes searx","limit":4}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one SearXNG search call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/search" {
		t.Fatalf("request = %s %s, want GET /search", req.Method, req.URL.Path)
	}
	if req.URL.Query().Get("q") != "gormes searx" || req.URL.Query().Get("format") != "json" || req.URL.Query().Get("categories") != "general" {
		t.Fatalf("query = %s, want SearXNG JSON params", req.URL.RawQuery)
	}

	var payload webSearchResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 1 || payload.Data.Web[0].Description != "SearXNG snippet" {
		t.Fatalf("output = %+v, want normalized SearXNG result", payload)
	}
}

func TestWebSearchToolCallsDuckDuckGoBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body: `<html><body>
<a rel="nofollow" class="result__a" href="/l/?kh=-1&uddg=https%3A%2F%2Fexample.test%2Fddg">Duck &amp; Go</a>
<a class="result__snippet" href="/l/?kh=-1&uddg=https%3A%2F%2Fexample.test%2Fddg">DDG snippet &amp; details</a>
</body></html>`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes ddg","limit":2}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one DuckDuckGo search call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/html" || req.URL.Query().Get("q") != "gormes ddg" {
		t.Fatalf("request = %s %s?%s, want GET /html?q=", req.Method, req.URL.Path, req.URL.RawQuery)
	}
	if got := req.Header.Get("User-Agent"); !strings.Contains(got, "Mozilla/5.0") {
		t.Fatalf("User-Agent = %q, want browser-ish user agent", got)
	}

	var payload webSearchResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 1 {
		t.Fatalf("output = %+v, want one normalized DuckDuckGo result", payload)
	}
	if got := payload.Data.Web[0]; got.Title != "Duck & Go" || got.URL != "https://example.test/ddg" || got.Description != "DDG snippet & details" {
		t.Fatalf("result = %+v, want decoded DDG title/url/snippet", got)
	}
}

func TestWebSearchToolFallsBackToDuckDuckGoLite(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{
		{status: http.StatusOK, body: `<html><body>No class-based results</body></html>`},
		{status: http.StatusOK, body: `<html><body>
<a rel="nofollow" href="/l/?kh=-1&uddg=https%3A%2F%2Fexample.test%2Flite" class="result-link">Lite Result</a>
<td class="result-snippet">Lite snippet</td>
</body></html>`},
	}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes lite","limit":2}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("requests = %d, want html then lite fallback", len(client.requests))
	}
	if client.requests[1].URL.Host != "lite.duckduckgo.com" {
		t.Fatalf("fallback host = %q, want lite.duckduckgo.com", client.requests[1].URL.Host)
	}
	var payload webSearchResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 1 {
		t.Fatalf("output = %+v, want one Lite result", payload)
	}
	if got := payload.Data.Web[0]; got.Title != "Lite Result" || got.URL != "https://example.test/lite" || got.Description != "Lite snippet" {
		t.Fatalf("result = %+v, want decoded Lite title/url/snippet", got)
	}
}

func TestWebExtractToolUsesDuckDuckGoInstantAnswer(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"Heading":"Planet IX","AbstractText":"Planet IX is a Web3 strategy game.","AbstractSource":"DuckDuckGo","AbstractURL":"https://planetix.com/"}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://planetix.com/"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one DuckDuckGo Instant Answer call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodGet || req.URL.Host != "api.duckduckgo.com" || req.URL.Query().Get("format") != "json" {
		t.Fatalf("request = %s %s?%s, want DuckDuckGo Instant Answer API", req.Method, req.URL.Host+req.URL.Path, req.URL.RawQuery)
	}
	var payload webExtractResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Planet IX" || payload.Results[0].Content != "Planet IX is a Web3 strategy game." || payload.Results[0].URL != "https://planetix.com/" {
		t.Fatalf("output = %+v, want Instant Answer extraction", payload)
	}
}

func TestWebExtractToolFetchesDuckDuckGoDirectTextDocument(t *testing.T) {
	const docsURL = "https://docs.openclaw.ai/llms.txt"
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		header: map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		body:   "# OpenClaw\n\nSelf-hosted gateway documentation index.\n\n- Telegram\n- WhatsApp\n",
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["`+docsURL+`"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one direct document fetch", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodGet || req.URL.String() != docsURL {
		t.Fatalf("request = %s %s, want direct GET %s", req.Method, req.URL.String(), docsURL)
	}
	if accept := req.Header.Get("Accept"); !strings.Contains(accept, "text/plain") || !strings.Contains(accept, "text/markdown") {
		t.Fatalf("Accept = %q, want direct text document media types", accept)
	}

	var payload webExtractResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("results len = %d, want one extracted document", len(payload.Results))
	}
	got := payload.Results[0]
	if got.URL != docsURL || got.Title != "llms.txt" || !strings.Contains(got.Content, "Self-hosted gateway documentation index") {
		t.Fatalf("result = %+v, want direct text document content", got)
	}
	if strings.Contains(got.Content, "No instant answer") {
		t.Fatalf("content = %q, want direct document content instead of DuckDuckGo fallback", got.Content)
	}
}

func TestWebExtractToolFetchesDuckDuckGoDirectHTMLDocument(t *testing.T) {
	const docsURL = "https://docs.openclaw.ai/concepts/presence"
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		header: map[string]string{"Content-Type": "text/html; charset=utf-8"},
		body: `<html>
<head><title>Presence - OpenClaw</title></head>
<body>
<nav>Navigation should not be extracted</nav>
<main>
  <h1>Presence</h1>
  <p>OpenClaw presence is a lightweight best-effort view.</p>
  <h2>Debugging tips</h2>
  <ul><li>Call <code>system-presence</code> against the Gateway.</li></ul>
  <p>See <a href="/concepts/typing-indicators">Typing indicators</a>.</p>
</main>
</body>
</html>`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["`+docsURL+`"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one direct document fetch", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodGet || req.URL.String() != docsURL {
		t.Fatalf("request = %s %s, want direct GET %s", req.Method, req.URL.String(), docsURL)
	}
	if accept := req.Header.Get("Accept"); !strings.Contains(accept, "text/html") || !strings.Contains(accept, "text/plain") {
		t.Fatalf("Accept = %q, want direct HTML/text media types", accept)
	}

	var payload webExtractResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("results len = %d, want one extracted document", len(payload.Results))
	}
	got := payload.Results[0]
	for _, want := range []string{"# Presence", "OpenClaw presence is a lightweight", "## Debugging tips", "- Call system-presence", "Typing indicators (https://docs.openclaw.ai/concepts/typing-indicators)"} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("content missing %q:\n%s", want, got.Content)
		}
	}
	if strings.Contains(got.Content, "Navigation should not be extracted") || strings.Contains(got.Content, "No instant answer") {
		t.Fatalf("content used nav or Instant Answer fallback:\n%s", got.Content)
	}
	if got.URL != docsURL || got.Title != "Presence - OpenClaw" {
		t.Fatalf("result metadata = %+v, want URL/title from direct HTML", got)
	}
}

func TestWebExtractToolDefaultDirectTextClientBlocksPrivateRedirects(t *testing.T) {
	tool := NewWebExtractTool(WebToolsConfig{
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
	}).(*webTool)
	client, ok := tool.directTextDocumentHTTPClient().(*http.Client)
	if !ok || client.CheckRedirect == nil {
		t.Fatalf("directTextDocumentHTTPClient() = %T, want default *http.Client with redirect guard", tool.directTextDocumentHTTPClient())
	}
	redirect, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/secret.txt", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if err := client.CheckRedirect(redirect, []*http.Request{{}}); err == nil {
		t.Fatal("CheckRedirect allowed private redirect, want blocked")
	}
}

func TestWebSearchToolCallsPerplexityBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"choices":[{"message":{"content":"Perplexity answer"}}],"citations":["https://example.test/one","https://example.test/two"]}`,
	}}}
	tool := NewWebSearchTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendPerplexity,
			BaseURL:   "https://api.perplexity.test",
			APIKey:    "perplexity-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes perplexity","limit":2}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Perplexity search call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/chat/completions" {
		t.Fatalf("request = %s %s, want POST /chat/completions", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer perplexity-secret" {
		t.Fatalf("Authorization = %q, want Perplexity bearer token", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["model"] != "sonar" || sent["max_tokens"] != float64(1000) {
		t.Fatalf("perplexity body = %#v, want sonar request defaults", sent)
	}

	var payload webSearchResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if !payload.Success || len(payload.Data.Web) != 2 {
		t.Fatalf("output = %+v, want citation-backed Perplexity results", payload)
	}
	if payload.Data.Web[0].URL != "https://example.test/one" || payload.Data.Web[0].Description != "Perplexity answer" {
		t.Fatalf("first result = %+v, want answer attached to citation", payload.Data.Web[0])
	}
}

func TestWebExtractToolCallsFirecrawlAndBlocksPrivateURLs(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":{"markdown":"# Example","metadata":{"title":"Example","sourceURL":"https://example.test/page"}}}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["http://127.0.0.1/admin","https://example.test/page"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want only public URL fetched", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/v2/scrape" {
		t.Fatalf("request = %s %s, want POST /v2/scrape", req.Method, req.URL.Path)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["url"] != "https://example.test/page" {
		t.Fatalf("scrape body url = %v, want public URL", sent["url"])
	}

	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
			Error   string `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 2 {
		t.Fatalf("results len = %d, want blocked + fetched result", len(payload.Results))
	}
	if !strings.Contains(payload.Results[0].Error, string(WebEvidencePrivateURLBlocked)) {
		t.Fatalf("blocked result error = %q, want private URL evidence", payload.Results[0].Error)
	}
	if payload.Results[1].URL != "https://example.test/page" || payload.Results[1].Title != "Example" || payload.Results[1].Content != "# Example" {
		t.Fatalf("fetched result = %+v, want normalized Firecrawl page", payload.Results[1])
	}
}

func TestWebExtractToolBlocksEmbeddedSecretsBeforeHTTP(t *testing.T) {
	client := &recordingWebHTTPClient{}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/?token=sk-secret-123456"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("requests = %d, want no provider call for secret-bearing URL", len(client.requests))
	}
	var payload struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if payload.Success || !strings.Contains(payload.Error, "contains what appears to be an API key or token") {
		t.Fatalf("payload = %+v, want blocked secret error", payload)
	}
}

func TestWebExtractToolAppliesWebsitePolicyBeforeFetch(t *testing.T) {
	client := &recordingWebHTTPClient{}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Policy: WebWebsitePolicy{
			Enabled: true,
			Domains: []string{"blocked.test"},
		},
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://blocked.test/page"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("requests = %d, want policy block before provider call", len(client.requests))
	}
	var payload struct {
		Results []struct {
			URL             string            `json:"url"`
			Content         string            `json:"content"`
			Error           string            `json:"error"`
			BlockedByPolicy map[string]string `json:"blocked_by_policy"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("results len = %d, want one policy block", len(payload.Results))
	}
	got := payload.Results[0]
	if got.URL != "https://blocked.test/page" || got.Content != "" || !strings.Contains(got.Error, "Blocked by website policy") {
		t.Fatalf("policy result = %+v, want blocked page", got)
	}
	if got.BlockedByPolicy["rule"] != "blocked.test" || got.BlockedByPolicy["source"] != "config" {
		t.Fatalf("blocked_by_policy = %#v, want rule/source", got.BlockedByPolicy)
	}
}

func TestWebExtractToolBlocksRedirectedFinalURL(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":{"markdown":"secret page","metadata":{"title":"Redirected","sourceURL":"https://blocked.test/final"}}}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Policy: WebWebsitePolicy{
			Enabled: true,
			Domains: []string{"blocked.test"},
		},
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://allowed.test/start"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want initial public URL fetched once", len(client.requests))
	}
	var payload struct {
		Results []struct {
			URL             string            `json:"url"`
			Title           string            `json:"title"`
			Content         string            `json:"content"`
			Error           string            `json:"error"`
			BlockedByPolicy map[string]string `json:"blocked_by_policy"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("results len = %d, want one redirected block", len(payload.Results))
	}
	got := payload.Results[0]
	if got.URL != "https://blocked.test/final" || got.Title != "Redirected" || got.Content != "" {
		t.Fatalf("redirect result = %+v, want final URL with empty content", got)
	}
	if !strings.Contains(got.Error, "Blocked by website policy") || got.BlockedByPolicy["rule"] != "blocked.test" {
		t.Fatalf("redirect policy = %+v, want blocked_by_policy", got)
	}
}

func TestWebExtractToolRemovesBase64ImagesFromOutput(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":{"markdown":"before (data:image/png;base64,QUJDREVGRw==) after data:image/jpeg;base64,SElKS0w=","metadata":{"title":"Image Data","sourceURL":"https://example.test/page"}}}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/page"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "QUJDREVGRw") || strings.Contains(text, "SElKS0w") {
		t.Fatalf("output leaked base64 image data: %s", text)
	}
	if count := strings.Count(text, "[BASE64_IMAGE_REMOVED]"); count != 2 {
		t.Fatalf("base64 placeholder count = %d, want 2 in output %s", count, text)
	}
}

func TestWebExtractToolProcessesLongContentWithProcessor(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":{"markdown":"0123456789abcdef","metadata":{"title":"Long Page","sourceURL":"https://example.test/long"}}}`,
	}}}
	processor := &recordingWebContentProcessor{summary: "processed summary"}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Processing: WebContentProcessingConfig{
			Enabled:        true,
			MinLength:      10,
			MaxInputChars:  1000,
			MaxOutputChars: 100,
		},
		ContentProcessor: processor,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/long"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(processor.requests) != 1 {
		t.Fatalf("processor requests = %d, want one long-content processing request", len(processor.requests))
	}
	if processor.requests[0].URL != "https://example.test/long" || processor.requests[0].Title != "Long Page" {
		t.Fatalf("processor request = %+v, want URL/title", processor.requests[0])
	}
	var payload struct {
		Results []struct {
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if got := payload.Results[0].Content; got != "processed summary" {
		t.Fatalf("content = %q, want processed summary", got)
	}
}

func TestWebCrawlToolCallsFirecrawlAndNormalizesPages(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":[{"markdown":"# Page 1","metadata":{"title":"Page 1","sourceURL":"https://example.test/docs"}},{"html":"<p>Page 2</p>","metadata":{"title":"Page 2","url":"https://example.test/docs/two"}}]}`,
	}}}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"example.test/docs","instructions":"Find docs","depth":"advanced"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Firecrawl crawl call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/v2/crawl" {
		t.Fatalf("request = %s %s, want POST /v2/crawl", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer fire-secret" {
		t.Fatalf("Authorization = %q, want bearer token", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["url"] != "https://example.test/docs" || sent["limit"] != float64(20) {
		t.Fatalf("crawl body = %#v, want prefixed url and Firecrawl page limit", sent)
	}
	if _, ok := sent["instructions"]; ok {
		t.Fatalf("crawl body = %#v, want Firecrawl instructions ignored like Hermes", sent)
	}
	scrapeOptions, _ := sent["scrape_options"].(map[string]any)
	formats, _ := scrapeOptions["formats"].([]any)
	if len(formats) != 1 || formats[0] != "markdown" {
		t.Fatalf("scrape_options = %#v, want markdown format", scrapeOptions)
	}

	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 2 {
		t.Fatalf("results len = %d, want 2", len(payload.Results))
	}
	if payload.Results[0].URL != "https://example.test/docs" || payload.Results[0].Title != "Page 1" || payload.Results[0].Content != "# Page 1" {
		t.Fatalf("first result = %+v, want normalized markdown page", payload.Results[0])
	}
	if payload.Results[1].URL != "https://example.test/docs/two" || payload.Results[1].Content != "<p>Page 2</p>" {
		t.Fatalf("second result = %+v, want html fallback page", payload.Results[1])
	}
}

func TestWebCrawlToolCallsTavilyWithInstructions(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Tavily Page","url":"https://example.test/page","raw_content":"Tavily crawl content"}]}`,
	}}}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendTavily,
			BaseURL:   "https://tavily.test",
			APIKey:    "tavily-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.test","instructions":"Find docs","depth":"advanced"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Tavily crawl call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crawl" {
		t.Fatalf("request = %s %s, want POST /crawl", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want Tavily key in JSON body", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["url"] != "https://example.test" || sent["api_key"] != "tavily-secret" {
		t.Fatalf("crawl body = %#v, want url and api_key", sent)
	}
	if sent["limit"] != float64(20) || sent["extract_depth"] != "advanced" || sent["instructions"] != "Find docs" {
		t.Fatalf("crawl body = %#v, want limit, advanced depth, and instructions", sent)
	}

	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].URL != "https://example.test/page" || payload.Results[0].Title != "Tavily Page" || payload.Results[0].Content != "Tavily crawl content" {
		t.Fatalf("output = %+v, want normalized Tavily crawl result", payload)
	}
}

func TestWebCrawlToolBlocksEmbeddedSecretsBeforeHTTP(t *testing.T) {
	client := &recordingWebHTTPClient{}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.test/?token=sk-secret-123456"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("requests = %d, want no provider call for secret-bearing crawl URL", len(client.requests))
	}
	var payload struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if payload.Success || !strings.Contains(payload.Error, "contains what appears to be an API key or token") {
		t.Fatalf("payload = %+v, want blocked secret error", payload)
	}
}

func TestWebCrawlToolAppliesWebsitePolicyBeforeFetch(t *testing.T) {
	client := &recordingWebHTTPClient{}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Policy: WebWebsitePolicy{
			Enabled: true,
			Domains: []string{"blocked.test"},
		},
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://blocked.test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("requests = %d, want policy block before provider call", len(client.requests))
	}
	var payload struct {
		Results []struct {
			URL             string            `json:"url"`
			Content         string            `json:"content"`
			Error           string            `json:"error"`
			BlockedByPolicy map[string]string `json:"blocked_by_policy"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("results len = %d, want one policy block", len(payload.Results))
	}
	got := payload.Results[0]
	if got.URL != "https://blocked.test" || got.Content != "" || !strings.Contains(got.Error, "Blocked by website policy") {
		t.Fatalf("policy result = %+v, want blocked crawl URL", got)
	}
	if got.BlockedByPolicy["rule"] != "blocked.test" || got.BlockedByPolicy["source"] != "config" {
		t.Fatalf("blocked_by_policy = %#v, want rule/source", got.BlockedByPolicy)
	}
}

func TestWebCrawlToolBlocksRedirectedFinalURL(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":[{"markdown":"secret crawl content","metadata":{"title":"Redirected crawl page","sourceURL":"https://blocked.test/final"}}]}`,
	}}}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Policy: WebWebsitePolicy{
			Enabled: true,
			Domains: []string{"blocked.test"},
		},
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://allowed.test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want initial public URL fetched once", len(client.requests))
	}
	var payload struct {
		Results []struct {
			URL             string            `json:"url"`
			Title           string            `json:"title"`
			Content         string            `json:"content"`
			Error           string            `json:"error"`
			BlockedByPolicy map[string]string `json:"blocked_by_policy"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	got := payload.Results[0]
	if got.URL != "https://blocked.test/final" || got.Title != "Redirected crawl page" || got.Content != "" {
		t.Fatalf("redirect result = %+v, want final URL with empty content", got)
	}
	if !strings.Contains(got.Error, "Blocked by website policy") || got.BlockedByPolicy["rule"] != "blocked.test" {
		t.Fatalf("redirect policy = %+v, want blocked_by_policy", got)
	}
}

func TestWebCrawlToolRemovesBase64ImagesFromOutput(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":[{"markdown":"before data:image/png;base64,QUJDREVGRw== after","metadata":{"title":"Image Data","sourceURL":"https://example.test/page"}}]}`,
	}}}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "QUJDREVGRw") {
		t.Fatalf("output leaked base64 image data: %s", text)
	}
	if !strings.Contains(text, "[BASE64_IMAGE_REMOVED]") {
		t.Fatalf("output = %s, want base64 placeholder", text)
	}
}

func TestWebCrawlToolProcessesLongContentWithProcessor(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"success":true,"data":[{"markdown":"0123456789abcdef","metadata":{"title":"Long Crawl Page","sourceURL":"https://example.test/long"}}]}`,
	}}}
	processor := &recordingWebContentProcessor{summary: "crawl summary"}
	tool := NewWebCrawlTool(WebToolsConfig{
		Client: client,
		Processing: WebContentProcessingConfig{
			Enabled:        true,
			MinLength:      10,
			MaxInputChars:  1000,
			MaxOutputChars: 100,
		},
		ContentProcessor: processor,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(processor.requests) != 1 {
		t.Fatalf("processor requests = %d, want one crawl page processing request", len(processor.requests))
	}
	if processor.requests[0].URL != "https://example.test/long" || processor.requests[0].Title != "Long Crawl Page" {
		t.Fatalf("processor request = %+v, want URL/title", processor.requests[0])
	}
	var payload struct {
		Results []struct {
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if got := payload.Results[0].Content; got != "crawl summary" {
		t.Fatalf("content = %q, want processed crawl summary", got)
	}
}

func TestWebExtractToolCallsTavilyBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Tavily Page","url":"https://example.test/tavily","content":"Tavily content"}]}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendTavily,
			BaseURL:   "https://tavily.test",
			APIKey:    "tavily-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/tavily"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Tavily extract call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/extract" {
		t.Fatalf("request = %s %s, want POST /extract", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want Tavily key in JSON body for Hermes parity", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["api_key"] != "tavily-secret" || sent["include_images"] != false {
		t.Fatalf("tavily extract body = %#v, want api_key and include_images=false", sent)
	}
	urls, _ := sent["urls"].([]any)
	if len(urls) != 1 || urls[0] != "https://example.test/tavily" {
		t.Fatalf("urls = %#v, want requested URL", sent["urls"])
	}

	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Tavily Page" || payload.Results[0].Content != "Tavily content" {
		t.Fatalf("output = %+v, want normalized Tavily page", payload)
	}
}

func TestWebExtractToolCallsExaBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Exa Page","url":"https://example.test/exa","text":"Exa text"}]}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendExa,
			BaseURL:   "https://exa.test",
			APIKey:    "exa-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/exa"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Exa contents call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/contents" {
		t.Fatalf("request = %s %s, want POST /contents", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("x-api-key"); got != "exa-secret" {
		t.Fatalf("x-api-key = %q, want Exa key", got)
	}
	if got := req.Header.Get("x-exa-integration"); got != "hermes-agent" {
		t.Fatalf("x-exa-integration = %q, want hermes-agent", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["text"] != true {
		t.Fatalf("exa contents body = %#v, want text=true", sent)
	}
	urls, _ := sent["urls"].([]any)
	if len(urls) != 1 || urls[0] != "https://example.test/exa" {
		t.Fatalf("urls = %#v, want requested URL", sent["urls"])
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Exa Page" || payload.Results[0].Content != "Exa text" {
		t.Fatalf("output = %+v, want normalized Exa page", payload)
	}
}

func TestWebExtractToolCallsParallelBackend(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusOK,
		body:   `{"results":[{"title":"Parallel Page","url":"https://example.test/parallel","full_content":"Parallel full content"}]}`,
	}}}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendParallel,
			BaseURL:   "https://parallel.test",
			APIKey:    "parallel-secret",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/parallel"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want one Parallel extract call", len(client.requests))
	}
	req := client.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/v1beta/extract" {
		t.Fatalf("request = %s %s, want POST /v1beta/extract", req.Method, req.URL.Path)
	}
	if got := req.Header.Get("x-api-key"); got != "parallel-secret" {
		t.Fatalf("x-api-key = %q, want Parallel key", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(client.bodies[0], &sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if sent["full_content"] != true {
		t.Fatalf("parallel extract body = %#v, want full_content=true", sent)
	}
	urls, _ := sent["urls"].([]any)
	if len(urls) != 1 || urls[0] != "https://example.test/parallel" {
		t.Fatalf("urls = %#v, want requested URL", sent["urls"])
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Parallel Page" || payload.Results[0].Content != "Parallel full content" {
		t.Fatalf("output = %+v, want normalized Parallel page", payload)
	}
}

func TestWebExtractToolFallsBackToCDPWhenFirecrawlFails(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusBadGateway,
		body:   `{"error":"scraper blocked"}`,
	}}}
	runner := &recordingHarnessRunner{
		result: BrowserHarnessProcessResult{Stdout: []byte(`{"schema_version":"gormes.browser.action.v1","evidence":"go_browser_harness_action_accepted","kind":"navigate","url":"https://example.test/fallback","title":"Fallback Page","text":"CDP rendered content"}`)},
	}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			APIKey:    "fire-secret",
			Available: true,
		},
		Browser: BrowserHarnessToolsConfig{
			Runner: runner,
			Env:    map[string]string{"CHROME_REMOTE_DEBUGGING_URL": "http://127.0.0.1:9222"},
			Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/fallback"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want Firecrawl attempted once before fallback", len(client.requests))
	}
	if len(runner.argv) < 3 {
		t.Fatalf("argv = %v, want go-browser-harness fallback call", runner.argv)
	}
	if got, wantPrefix := strings.Join(runner.argv[:2], "\x00"), "go-browser-harness\x00--action-json"; got != wantPrefix {
		t.Fatalf("argv prefix = %q, want %q", got, wantPrefix)
	}
	action := decodeHarnessAction(t, runner.argv[2])
	if action.Kind != BrowserActionNavigate || action.URL != "https://example.test/fallback" || !action.NewTab {
		t.Fatalf("fallback action = %#v, want new-tab navigate to requested URL", action)
	}

	var payload struct {
		Results []struct {
			URL      string      `json:"url"`
			Title    string      `json:"title"`
			Content  string      `json:"content"`
			Error    string      `json:"error"`
			Evidence WebEvidence `json:"evidence"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].URL != "https://example.test/fallback" || payload.Results[0].Title != "Fallback Page" || payload.Results[0].Content != "CDP rendered content" {
		t.Fatalf("output = %+v, want CDP-rendered fallback page", payload)
	}
	if payload.Results[0].Error != "" || payload.Results[0].Evidence != "" {
		t.Fatalf("fallback should replace provider error, got %+v", payload.Results[0])
	}
}

func TestWebExtractToolFallsBackToCDPWhenBatchProviderFails(t *testing.T) {
	client := &recordingWebHTTPClient{responses: []recordedWebResponse{{
		status: http.StatusServiceUnavailable,
		body:   `{"error":"provider unavailable"}`,
	}}}
	runner := &recordingHarnessRunner{
		result: BrowserHarnessProcessResult{Stdout: []byte(`{"schema_version":"gormes.browser.action.v1","evidence":"go_browser_harness_action_accepted","kind":"navigate","url":"https://example.test/exa-fallback","title":"Exa Fallback","text":"CDP batch fallback"}`)},
	}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendExa,
			BaseURL:   "https://exa.test",
			APIKey:    "exa-secret",
			Available: true,
		},
		Browser: BrowserHarnessToolsConfig{
			Runner: runner,
			Env:    map[string]string{"CHROME_REMOTE_DEBUGGING_URL": "http://127.0.0.1:9222"},
			Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/exa-fallback"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want Exa attempted once before fallback", len(client.requests))
	}
	if len(runner.argv) < 3 {
		t.Fatalf("argv = %v, want go-browser-harness fallback call", runner.argv)
	}
	action := decodeHarnessAction(t, runner.argv[2])
	if action.Kind != BrowserActionNavigate || action.URL != "https://example.test/exa-fallback" || !action.NewTab {
		t.Fatalf("fallback action = %#v, want new-tab navigate to requested URL", action)
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			Error   string `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Exa Fallback" || payload.Results[0].Content != "CDP batch fallback" || payload.Results[0].Error != "" {
		t.Fatalf("output = %+v, want CDP-rendered batch fallback page", payload)
	}
}

func TestWebExtractToolUsesCDPForDuckDuckGoURLExtractWhenConfigured(t *testing.T) {
	client := &recordingWebHTTPClient{}
	runner := &recordingHarnessRunner{
		result: BrowserHarnessProcessResult{Stdout: []byte(`{"schema_version":"gormes.browser.action.v1","evidence":"go_browser_harness_action_accepted","kind":"navigate","url":"https://example.test/ddg-cdp","title":"Rendered DDG Fallback","text":"browser rendered page"}`)},
	}
	tool := NewWebExtractTool(WebToolsConfig{
		Client: client,
		Resolution: WebBackendResolution{
			Backend:   WebBackendDuckDuckGo,
			BaseURL:   webDefaultDuckDuckGoBaseURL,
			Available: true,
		},
		Browser: BrowserHarnessToolsConfig{
			Runner: runner,
			Env:    map[string]string{"BROWSER_CDP_URL": "http://127.0.0.1:9223"},
			Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/ddg-cdp"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("requests = %d, want DuckDuckGo Instant Answer skipped when CDP extract fallback is configured", len(client.requests))
	}
	if runner.env["CHROME_REMOTE_DEBUGGING_URL"] != "http://127.0.0.1:9223" || runner.env["BROWSER_CDP_URL"] != "http://127.0.0.1:9223" {
		t.Fatalf("runner env = %+v, want both CDP env aliases populated", runner.env)
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Rendered DDG Fallback" || payload.Results[0].Content != "browser rendered page" {
		t.Fatalf("payload = %+v, want browser-rendered extract", payload)
	}
}

func TestWebExtractToolCallsCDPBackendThroughBrowserHarness(t *testing.T) {
	runner := &recordingHarnessRunner{
		result: BrowserHarnessProcessResult{Stdout: []byte(`{"schema_version":"gormes.browser.action.v1","evidence":"go_browser_harness_action_accepted","kind":"navigate","url":"https://example.test/page","title":"Example Page","text":"Rendered content"}`)},
	}
	tool := NewWebExtractTool(WebToolsConfig{
		Resolution: WebBackendResolution{
			Backend:   WebBackendCDP,
			BaseURL:   "http://127.0.0.1:9222",
			Available: true,
		},
		Browser: BrowserHarnessToolsConfig{
			Runner: runner,
			Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/page"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, wantPrefix := strings.Join(runner.argv[:2], "\x00"), "go-browser-harness\x00--action-json"; got != wantPrefix {
		t.Fatalf("argv prefix = %q, want %q", got, wantPrefix)
	}
	action := decodeHarnessAction(t, runner.argv[2])
	if action.SchemaVersion != browserHarnessActionSchemaVersion || action.Kind != BrowserActionNavigate || action.URL != "https://example.test/page" {
		t.Fatalf("CDP extract action = %#v", action)
	}
	if !action.NewTab {
		t.Fatalf("CDP extract must navigate in a new tab: %#v", action)
	}
	if runner.env["CHROME_REMOTE_DEBUGGING_URL"] != "http://127.0.0.1:9222" {
		t.Fatalf("CHROME_REMOTE_DEBUGGING_URL = %q, want resolution base URL", runner.env["CHROME_REMOTE_DEBUGGING_URL"])
	}
	if runner.env["BROWSER_CDP_URL"] != "http://127.0.0.1:9222" {
		t.Fatalf("BROWSER_CDP_URL = %q, want resolution base URL", runner.env["BROWSER_CDP_URL"])
	}

	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].URL != "https://example.test/page" || payload.Results[0].Title != "Example Page" || payload.Results[0].Content != "Rendered content" {
		t.Fatalf("output = %+v, want normalized CDP page", payload)
	}
}

func TestWebExtractToolBlocksPrivateURLBeforeCDPBackend(t *testing.T) {
	runner := &recordingHarnessRunner{}
	tool := NewWebExtractTool(WebToolsConfig{
		Resolution: WebBackendResolution{
			Backend:   WebBackendCDP,
			BaseURL:   "http://127.0.0.1:9222",
			Available: true,
		},
		Browser: BrowserHarnessToolsConfig{Runner: runner},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["http://127.0.0.1/admin"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(runner.argv) != 0 {
		t.Fatalf("argv = %v, want no browser harness call for private URL", runner.argv)
	}
	var payload struct {
		Results []struct {
			Error    string      `json:"error"`
			Evidence WebEvidence `json:"evidence"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Evidence != WebEvidencePrivateURLBlocked {
		t.Fatalf("payload = %+v, want private URL block evidence", payload)
	}
}

func TestWebSearchToolRejectsCDPBackend(t *testing.T) {
	tool := NewWebSearchTool(WebToolsConfig{
		Resolution: WebBackendResolution{
			Backend:   WebBackendCDP,
			BaseURL:   "http://127.0.0.1:9222",
			Available: true,
		},
	})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"gormes"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload webSearchResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	if payload.Success || payload.Evidence != WebEvidenceBackendUnsupported || !strings.Contains(payload.Error, "only supports web_extract") {
		t.Fatalf("payload = %+v, want CDP unsupported for web_search", payload)
	}
}

type recordingWebHTTPClient struct {
	responses []recordedWebResponse
	requests  []*http.Request
	bodies    [][]byte
}

type recordingWebContentProcessor struct {
	summary  string
	requests []WebContentProcessRequest
}

func (p *recordingWebContentProcessor) ProcessWebContent(_ context.Context, req WebContentProcessRequest) (string, error) {
	p.requests = append(p.requests, req)
	return p.summary, nil
}

type recordedWebResponse struct {
	status int
	header map[string]string
	body   string
}

func (c *recordingWebHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}
	c.requests = append(c.requests, req.Clone(req.Context()))
	c.bodies = append(c.bodies, body)
	idx := len(c.requests) - 1
	resp := recordedWebResponse{status: http.StatusOK, body: `{}`}
	if idx < len(c.responses) {
		resp = c.responses[idx]
	}
	if resp.status == 0 {
		resp.status = http.StatusOK
	}
	header := make(http.Header)
	for k, v := range resp.header {
		header.Set(k, v)
	}
	return &http.Response{
		StatusCode: resp.status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
	}, nil
}
