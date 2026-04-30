package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWebLiveProviderSmoke(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GORMES_LIVE_WEB_SMOKE")) != "1" {
		t.Skip("set GORMES_LIVE_WEB_SMOKE=1 with web provider credentials to run live web smoke")
	}

	resolved := ResolveWebBackendWithConfig(nil, WebBackendConfig{
		Backend:             os.Getenv("GORMES_LIVE_WEB_BACKEND"),
		UseGateway:          strings.TrimSpace(os.Getenv("GORMES_LIVE_WEB_USE_GATEWAY")) == "1",
		ManagedToolsEnabled: true,
		AuthStorePath:       os.Getenv("GORMES_LIVE_WEB_AUTH_STORE"),
	})
	if !resolved.Available {
		t.Fatalf("live web smoke requested but no web backend is available: backend=%s evidence=%s source=%s", resolved.Backend, resolved.Evidence, resolved.Source)
	}

	cfg := WebToolsConfig{
		Resolution: resolved,
		Timeout:    45 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	searchOut, err := NewWebSearchTool(cfg).Execute(ctx, json.RawMessage(`{"query":"example domain","limit":1}`))
	if err != nil {
		t.Fatalf("live web_search failed: %v", err)
	}
	var searchPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Web []webSearchResult `json:"web"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(searchOut, &searchPayload); err != nil {
		t.Fatalf("decode live web_search output: %v", err)
	}
	if !searchPayload.Success || len(searchPayload.Data.Web) == 0 {
		t.Fatalf("live web_search returned no results: success=%t error=%q", searchPayload.Success, searchPayload.Error)
	}

	extractOut, err := NewWebExtractTool(cfg).Execute(ctx, json.RawMessage(`{"urls":["https://example.com"]}`))
	if err != nil {
		t.Fatalf("live web_extract failed: %v", err)
	}
	var extractPayload webExtractResponse
	if err := json.Unmarshal(extractOut, &extractPayload); err != nil {
		t.Fatalf("decode live web_extract output: %v", err)
	}
	if len(extractPayload.Results) == 0 || extractPayload.Results[0].Content == "" {
		t.Fatalf("live web_extract returned no content: %+v", extractPayload.Results)
	}

	if strings.TrimSpace(os.Getenv("GORMES_LIVE_WEB_CRAWL_SMOKE")) != "1" {
		t.Log("set GORMES_LIVE_WEB_CRAWL_SMOKE=1 to include live web_crawl")
		return
	}
	if resolved.Backend != WebBackendFirecrawl && resolved.Backend != WebBackendTavily {
		t.Skipf("web_crawl smoke requires Firecrawl or Tavily, got %s", resolved.Backend)
	}
	crawlOut, err := NewWebCrawlTool(cfg).Execute(ctx, json.RawMessage(`{"url":"https://example.com","depth":"basic"}`))
	if err != nil {
		t.Fatalf("live web_crawl failed: %v", err)
	}
	var crawlPayload webExtractResponse
	if err := json.Unmarshal(crawlOut, &crawlPayload); err != nil {
		t.Fatalf("decode live web_crawl output: %v", err)
	}
	if len(crawlPayload.Results) == 0 {
		t.Fatalf("live web_crawl returned no pages")
	}
}
