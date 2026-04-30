package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWebToolsExposeHermesNamesAndSchemas(t *testing.T) {
	webTools := NewWebTools(WebToolsConfig{
		Resolution: WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   "https://firecrawl.test",
			Available: true,
		},
	})
	if len(webTools) != 2 {
		t.Fatalf("NewWebTools len = %d, want 2", len(webTools))
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
	for _, name := range []string{WebToolSearch, WebToolExtract} {
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

	missing := ResolveWebBackend(map[string]string{})
	if missing.Available {
		t.Fatalf("empty env resolved as available: %+v", missing)
	}
	if missing.Evidence != WebEvidenceProviderUnavailable {
		t.Fatalf("missing Evidence = %q, want %q", missing.Evidence, WebEvidenceProviderUnavailable)
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
	body   string
}

func (c *recordingWebHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
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
	return &http.Response{
		StatusCode: resp.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(resp.body)),
	}, nil
}
