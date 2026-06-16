package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goscrapling"
	goscraplingbrowser "github.com/TrebuchetDynamics/goscrapling/engines/browser"
)

func NewWebTools(cfg WebToolsConfig) []Tool {
	return []Tool{
		NewWebCrawlTool(cfg),
		NewWebExtractTool(cfg),
		NewWebSearchTool(cfg),
	}
}

func NewWebSearchTool(cfg WebToolsConfig) Tool {
	return &webTool{
		name:   WebToolSearch,
		desc:   "Search the web for information on any topic. Returns up to 5 relevant results with titles, URLs, and descriptions.",
		schema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"The search query to look up on the web. You may include backend-supported operators such as site:example.com, filetype:pdf, intitle:word, -term, or \"exact phrase\"."},"limit":{"type":"integer","description":"Maximum number of results to return. Defaults to 5.","minimum":1,"maximum":100,"default":5}},"required":["query"]}`),
		cfg:    normalizeWebToolsConfig(cfg),
	}
}

func NewWebExtractTool(cfg WebToolsConfig) Tool {
	return &webTool{
		name:   WebToolExtract,
		desc:   "Extract content from web page URLs. Returns page content in markdown format. Also works with PDF URLs; pass the PDF link directly and it converts to markdown text.",
		schema: json.RawMessage(`{"type":"object","properties":{"urls":{"type":"array","items":{"type":"string"},"maxItems":5,"description":"List of URLs to extract content from (max 5 URLs per call)"},"css_selector":{"type":"string","description":"Optional CSS selector for local/static extraction with goscrapling. When set, Gormes fetches each public URL locally and returns only matching element text."}},"required":["urls"]}`),
		cfg:    normalizeWebToolsConfig(cfg),
	}
}

func NewWebCrawlTool(cfg WebToolsConfig) Tool {
	return &webTool{
		name:   WebToolCrawl,
		desc:   "Crawl a website and return markdown content for discovered pages. Use this for multi-page sites when web_extract is too broad or a page is too large.",
		schema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"Base URL to crawl. If no scheme is supplied, https:// is assumed."},"instructions":{"type":"string","description":"Optional focused crawling instructions. Tavily forwards these instructions; Firecrawl crawl ignores them because the crawl API does not support a prompt parameter."},"depth":{"type":"string","description":"Extraction depth for Tavily crawl. Defaults to basic.","enum":["basic","advanced"],"default":"basic"}},"required":["url"]}`),
		cfg:    normalizeWebToolsConfig(cfg),
	}
}

func (t *webTool) Name() string { return t.name }

func (t *webTool) Description() string { return t.desc }

func (t *webTool) Schema() json.RawMessage {
	return append(json.RawMessage(nil), t.schema...)
}

func (t *webTool) Timeout() time.Duration {
	return t.cfg.timeout() + 5*time.Second
}

func (t *webTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	switch t.name {
	case WebToolSearch:
		return t.executeSearch(ctx, args)
	case WebToolExtract:
		return t.executeExtract(ctx, args)
	case WebToolCrawl:
		return t.executeCrawl(ctx, args)
	default:
		return nil, fmt.Errorf("%s: unknown web tool", t.name)
	}
}

func (t *webTool) executeSearch(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("%s: invalid args: %w", t.name, err)
	}
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, fmt.Errorf("%s: query is required", t.name)
	}
	if !t.cfg.Resolution.Available {
		return webSearchFailure(webProviderUnavailableMessage(), WebEvidenceProviderUnavailable)
	}
	if t.cfg.Resolution.Backend == WebBackendCDP || t.cfg.Resolution.Backend == WebBackendGoscraplingBrowser {
		return webSearchFailure("web_search requires an indexed search backend; CDP/browser backend only supports web_extract for known URLs.", WebEvidenceBackendUnsupported)
	}

	limit := in.Limit
	if limit <= 0 {
		limit = t.cfg.defaultLimit()
	}
	if limit > t.cfg.maxSearch() {
		limit = t.cfg.maxSearch()
	}
	raw, err := t.executeSearchBackend(ctx, query, limit)
	if err != nil {
		return webSearchFailure("Error searching web: "+redactWebError(err.Error(), t.cfg.Resolution.APIKey), WebEvidenceRequestFailed)
	}
	response := normalizeWebSearch(raw)
	response.Backend = t.cfg.Resolution.Backend
	response.Source = t.cfg.Resolution.Source
	if t.cfg.Resolution.Backend == WebBackendPerplexity && webSearchResponseHasNoSourceURLs(response) {
		response.Degraded = true
		response.DegradedReason = "no citations returned; answer may be model-synthesized"
	}
	return json.Marshal(response)
}

func (t *webTool) executeSearchBackend(ctx context.Context, query string, limit int) (map[string]any, error) {
	switch t.cfg.Resolution.Backend {
	case WebBackendTavily:
		return t.postProviderJSON(ctx, "search", map[string]any{
			"query":               query,
			"max_results":         minInt(limit, 20),
			"include_raw_content": false,
			"include_images":      false,
			"api_key":             t.cfg.Resolution.APIKey,
		}, nil)
	case WebBackendExa:
		return t.postProviderJSON(ctx, "search", map[string]any{
			"query":      query,
			"numResults": limit,
			"contents": map[string]any{
				"highlights": true,
			},
		}, map[string]string{
			"x-api-key":         t.cfg.Resolution.APIKey,
			"x-exa-integration": "hermes-agent",
		})
	case WebBackendParallel:
		return t.postProviderJSON(ctx, "v1beta/search", map[string]any{
			"objective":      query,
			"search_queries": []string{query},
			"mode":           parallelSearchMode(),
			"max_results":    minInt(limit, 20),
		}, map[string]string{
			"x-api-key": t.cfg.Resolution.APIKey,
		})
	case WebBackendBrave:
		return t.executeBraveSearch(ctx, query, limit)
	case WebBackendSearXNG:
		return t.executeSearXNGSearch(ctx, query)
	case WebBackendPerplexity:
		return t.executePerplexitySearch(ctx, query, limit)
	case WebBackendDuckDuckGo:
		return t.executeDuckDuckGoSearch(ctx, query, limit)
	default:
		return t.postFirecrawlJSON(ctx, "search", map[string]any{
			"query": query,
			"limit": limit,
		})
	}
}

func (t *webTool) executeBraveSearch(ctx context.Context, query string, limit int) (map[string]any, error) {
	endpoint, err := webProviderEndpoint(t.cfg.Resolution.BaseURL, "res/v1/web/search")
	if err != nil {
		return nil, err
	}
	endpoint, err = webURLWithQuery(endpoint, url.Values{
		"q":     []string{query},
		"count": []string{fmt.Sprintf("%d", minInt(limit, 20))},
	})
	if err != nil {
		return nil, err
	}
	return t.getJSON(ctx, endpoint, map[string]string{
		"Accept":               "application/json",
		"X-Subscription-Token": t.cfg.Resolution.APIKey,
	})
}

func (t *webTool) executeSearXNGSearch(ctx context.Context, query string) (map[string]any, error) {
	endpoint, err := webProviderEndpoint(t.cfg.Resolution.BaseURL, "search")
	if err != nil {
		return nil, err
	}
	endpoint, err = webURLWithQuery(endpoint, url.Values{
		"q":          []string{query},
		"format":     []string{"json"},
		"categories": []string{"general"},
	})
	if err != nil {
		return nil, err
	}
	return t.getJSON(ctx, endpoint, map[string]string{"Accept": "application/json"})
}

func (t *webTool) executePerplexitySearch(ctx context.Context, query string, limit int) (map[string]any, error) {
	raw, err := t.postProviderJSON(ctx, "chat/completions", map[string]any{
		"model": "sonar",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a search assistant. Provide concise search results with source-backed facts. Do not add unsupported claims.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Search for: %s. Return up to %d relevant results.", query, minInt(limit, 20)),
			},
		},
		"max_tokens": 1000,
	}, map[string]string{
		"Authorization": "Bearer " + t.cfg.Resolution.APIKey,
		"User-Agent":    webDefaultUserAgent,
	})
	if err != nil {
		return nil, err
	}
	return normalizePerplexitySearch(raw), nil
}

func (t *webTool) executeDuckDuckGoSearch(ctx context.Context, query string, limit int) (map[string]any, error) {
	endpoint, err := webProviderEndpoint(t.cfg.Resolution.BaseURL, "html")
	if err != nil {
		return nil, err
	}
	endpoint, err = webURLWithQuery(endpoint, url.Values{"q": []string{query}})
	if err != nil {
		return nil, err
	}
	body, err := t.getRaw(ctx, endpoint, map[string]string{
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"User-Agent": webDefaultUserAgent,
	})
	if err != nil {
		return nil, err
	}
	results := duckDuckGoSearchResults(body, limit)
	if len(results) == 0 {
		liteEndpoint := strings.Replace(endpoint, "html.duckduckgo.com", "lite.duckduckgo.com", 1)
		liteBody, liteErr := t.getRaw(ctx, liteEndpoint, map[string]string{
			"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"User-Agent": webDefaultUserAgent,
		})
		if liteErr == nil {
			results = duckDuckGoLiteSearchResults(liteBody, limit)
		}
	}
	return map[string]any{"results": results}, nil
}

func (t *webTool) executeExtract(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		URLs        []string `json:"urls"`
		Format      string   `json:"format"`
		CSSSelector string   `json:"css_selector"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("%s: invalid args: %w", t.name, err)
	}
	if len(in.URLs) == 0 {
		return nil, fmt.Errorf("%s: urls is required", t.name)
	}
	if len(in.URLs) > t.cfg.maxExtract() {
		return nil, fmt.Errorf("%s: max %d urls per call", t.name, t.cfg.maxExtract())
	}
	if webURLsContainSecrets(in.URLs) {
		return json.Marshal(webErrorResponse{
			Success: false,
			Error:   "Blocked: URL contains what appears to be an API key or token. Secrets must not be sent in URLs.",
		})
	}
	if t.cfg.Resolution.Backend == WebBackendGoscraplingBrowser {
		if !t.cfg.Resolution.Available {
			return json.Marshal(webExtractResponse{Results: []webExtractResult{{
				Error:    webProviderUnavailableMessage(),
				Evidence: WebEvidenceProviderUnavailable,
			}}})
		}
		return t.executeGoscraplingBrowserExtract(ctx, in.URLs, in.CSSSelector)
	}
	if strings.TrimSpace(in.CSSSelector) != "" {
		return t.executeGoscraplingExtract(ctx, in.URLs, in.CSSSelector)
	}
	if !t.cfg.Resolution.Available {
		return json.Marshal(webExtractResponse{Results: []webExtractResult{{
			Error:    webProviderUnavailableMessage(),
			Evidence: WebEvidenceProviderUnavailable,
		}}})
	}
	if t.cfg.Resolution.Backend == WebBackendCDP {
		return t.executeCDPExtract(ctx, in.URLs, in.Format)
	}
	if t.cfg.Resolution.Backend == WebBackendDuckDuckGo && t.cdpFallbackAvailable() {
		return t.executeCDPExtract(ctx, in.URLs, in.Format)
	}
	if t.cfg.Resolution.Backend != WebBackendFirecrawl {
		return t.executeProviderExtract(ctx, in.URLs, in.Format)
	}

	results := make([]webExtractResult, 0, len(in.URLs))
	formats := webExtractFormats(in.Format)
	for _, rawURL := range in.URLs {
		trimmed := strings.TrimSpace(rawURL)
		if errResult, blocked := t.blockedWebExtractRequestResult(trimmed); blocked {
			results = append(results, errResult)
			continue
		}
		raw, err := t.postFirecrawlJSON(ctx, "scrape", map[string]any{
			"url":     trimmed,
			"formats": formats,
		})
		if err != nil {
			results = append(results, t.webExtractFailureOrCDPFallback(ctx, trimmed, "Error extracting web page: ", err))
			continue
		}
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(normalizeWebExtract(trimmed, in.Format, raw))))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
}

func (t *webTool) executeCDPExtract(ctx context.Context, urls []string, _ string) (json.RawMessage, error) {
	results := make([]webExtractResult, 0, len(urls))
	for _, rawURL := range urls {
		trimmed := strings.TrimSpace(rawURL)
		if errResult, blocked := t.blockedWebExtractRequestResult(trimmed); blocked {
			results = append(results, errResult)
			continue
		}
		result, err := t.runCDPExtractURL(ctx, trimmed)
		if err != nil {
			results = append(results, webExtractResult{
				URL:      trimmed,
				Error:    "Error extracting web page with CDP: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
				Evidence: WebEvidenceRequestFailed,
			})
			continue
		}
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result)))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
}

func (t *webTool) runCDPExtractURL(ctx context.Context, rawURL string) (webExtractResult, error) {
	actionReq := BrowserHarnessActionRequest{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Kind:          BrowserActionNavigate,
		TaskID:        WebToolExtract,
		URL:           rawURL,
		NewTab:        true,
	}
	actionJSON, err := json.Marshal(actionReq)
	if err != nil {
		return webExtractResult{}, fmt.Errorf("web: marshal CDP action: %w", err)
	}

	cfg := t.cdpBrowserConfig()
	bridge := BrowserHarnessBridge{
		Command:  cfg.Command,
		Protocol: cfg.Protocol,
		Runner:   cfg.Runner,
		Backend:  cfg.Backend,
	}
	commandResult, err := bridge.Run(ctx, BrowserHarnessCommandRequest{
		Command:    cfg.Command,
		Protocol:   cfg.Protocol,
		ActionJSON: actionJSON,
		TaskID:     WebToolExtract,
		Action: BrowserAction{
			Kind:   BrowserActionNavigate,
			TaskID: WebToolExtract,
			URL:    rawURL,
		},
		Backend:   cfg.Backend,
		Env:       cfg.Env,
		Timeout:   cfg.Timeout,
		MediaType: firstNonEmpty(cfg.MediaType, "application/json"),
		Budget:    cfg.Budget,
	})
	if err != nil {
		return webExtractResult{}, err
	}
	if commandResult.Evidence != BrowserHarnessEvidenceCommandOK {
		return webExtractResult{}, fmt.Errorf("browser harness evidence: %s", commandResult.Evidence)
	}

	var payload struct {
		URL      string `json:"url"`
		Title    string `json:"title"`
		Text     string `json:"text"`
		Message  string `json:"message"`
		Evidence string `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(commandResult.Envelope.Text), &payload); err != nil {
		return webExtractResult{}, fmt.Errorf("decode browser harness result: %w", err)
	}
	if payload.Message != "" && payload.Text == "" && payload.Title == "" {
		return webExtractResult{}, fmt.Errorf("%s", payload.Message)
	}
	return webExtractResult{
		URL:     firstNonEmpty(payload.URL, rawURL),
		Title:   payload.Title,
		Content: payload.Text,
	}, nil
}

func (t *webTool) cdpBrowserConfig() BrowserHarnessToolsConfig {
	cfg := cloneBrowserHarnessToolsConfig(t.cfg.Browser)
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	if endpoint := t.cdpEndpoint(); endpoint != "" {
		if strings.TrimSpace(cfg.Env["CHROME_REMOTE_DEBUGGING_URL"]) == "" {
			cfg.Env["CHROME_REMOTE_DEBUGGING_URL"] = endpoint
		}
		if strings.TrimSpace(cfg.Env["BROWSER_CDP_URL"]) == "" {
			cfg.Env["BROWSER_CDP_URL"] = endpoint
		}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = t.cfg.timeout()
	}
	if cfg.MediaType == "" {
		cfg.MediaType = "text/plain"
	}
	if cfg.Budget.TextBudgetBytes <= 0 {
		cfg.Budget.TextBudgetBytes = maxWebResponseBytes
	}
	if cfg.Budget.PreviewBytes <= 0 {
		cfg.Budget.PreviewBytes = maxWebResponseBytes
	}
	return cfg
}

func (t *webTool) cdpEndpoint() string {
	if t.cfg.Browser.Env != nil {
		if endpoint := strings.TrimSpace(t.cfg.Browser.Env["CHROME_REMOTE_DEBUGGING_URL"]); endpoint != "" {
			return endpoint
		}
		if endpoint := strings.TrimSpace(t.cfg.Browser.Env["BROWSER_CDP_URL"]); endpoint != "" {
			return endpoint
		}
	}
	if t.cfg.Resolution.Backend == WebBackendCDP {
		if endpoint := strings.TrimSpace(t.cfg.Resolution.BaseURL); endpoint != "" {
			return endpoint
		}
	}
	return strings.TrimSpace(firstNonEmpty(os.Getenv("CHROME_REMOTE_DEBUGGING_URL"), os.Getenv("BROWSER_CDP_URL")))
}

func (t *webTool) cdpFallbackAvailable() bool {
	return t.cfg.Resolution.Backend != WebBackendCDP && t.cdpEndpoint() != ""
}

func (t *webTool) webExtractFailureOrCDPFallback(ctx context.Context, rawURL string, prefix string, err error) webExtractResult {
	if t.cdpFallbackAvailable() {
		result, cdpErr := t.runCDPExtractURL(ctx, rawURL)
		if cdpErr == nil {
			return t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result))
		}
		err = fmt.Errorf("%w; CDP fallback failed: %v", err, cdpErr)
	}
	return webExtractResult{
		URL:      rawURL,
		Error:    prefix + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
		Evidence: WebEvidenceRequestFailed,
	}
}

func (t *webTool) executeProviderExtract(ctx context.Context, urls []string, format string) (json.RawMessage, error) {
	blocked := make([]webExtractResult, 0)
	safeURLs := make([]string, 0, len(urls))
	for _, rawURL := range urls {
		trimmed := strings.TrimSpace(rawURL)
		if errResult, blockedURL := t.blockedWebExtractRequestResult(trimmed); blockedURL {
			blocked = append(blocked, errResult)
			continue
		}
		safeURLs = append(safeURLs, trimmed)
	}

	results := make([]webExtractResult, 0, len(blocked)+len(safeURLs))
	results = append(results, blocked...)
	if len(safeURLs) == 0 {
		return marshalWebExtractResponse(webExtractResponse{Results: results})
	}

	raw, err := t.executeExtractBackend(ctx, safeURLs)
	if err != nil {
		for _, url := range safeURLs {
			results = append(results, t.webExtractFailureOrCDPFallback(ctx, url, "Error extracting web page: ", err))
		}
		return marshalWebExtractResponse(webExtractResponse{Results: results})
	}
	for _, result := range normalizeWebExtractDocuments(safeURLs, format, raw) {
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result)))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
}

func (t *webTool) executeGoscraplingExtract(ctx context.Context, urls []string, selector string) (json.RawMessage, error) {
	selector = strings.TrimSpace(selector)
	results := make([]webExtractResult, 0, len(urls))
	for _, rawURL := range urls {
		trimmed := strings.TrimSpace(rawURL)
		if errResult, blocked := t.blockedWebExtractRequestResult(trimmed); blocked {
			results = append(results, errResult)
			continue
		}
		result, err := t.goscraplingExtractURL(ctx, trimmed, selector)
		if err != nil {
			results = append(results, webExtractResult{
				URL:      trimmed,
				Error:    "Error extracting web page with goscrapling: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
				Evidence: WebEvidenceRequestFailed,
			})
			continue
		}
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result)))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
}

func (t *webTool) executeGoscraplingBrowserExtract(ctx context.Context, urls []string, selector string) (json.RawMessage, error) {
	selector = strings.TrimSpace(selector)
	results := make([]webExtractResult, 0, len(urls))
	for _, rawURL := range urls {
		trimmed := strings.TrimSpace(rawURL)
		if errResult, blocked := t.blockedWebExtractRequestResult(trimmed); blocked {
			results = append(results, errResult)
			continue
		}
		result, err := t.goscraplingBrowserExtractURL(ctx, trimmed, selector)
		if err != nil {
			results = append(results, webExtractResult{
				URL:        trimmed,
				Error:      "Error extracting web page with goscrapling browser: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
				Evidence:   WebEvidenceRequestFailed,
				Extraction: t.goscraplingBrowserExtractionEvidence(trimmed, selector, nil),
			})
			continue
		}
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result)))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
}

func (t *webTool) goscraplingExtractURL(ctx context.Context, rawURL, selector string) (webExtractResult, error) {
	selector = strings.TrimSpace(selector)
	reqCtx, cancel := context.WithTimeout(ctx, t.cfg.timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return webExtractResult{}, err
	}
	req.Header.Set("Accept", "text/markdown,text/plain,text/html,application/xhtml+xml,text/*;q=0.9,application/json;q=0.8,application/xml;q=0.8,*/*;q=0.1")
	req.Header.Set("User-Agent", webDefaultUserAgent)

	resp, err := t.directTextDocumentHTTPClient().Do(req)
	if err != nil {
		return webExtractResult{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxWebResponseBytes))
	if err != nil {
		return webExtractResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return webExtractResult{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	if result, blocked := t.blockedWebExtractRequestResult(finalURL); blocked {
		return result, nil
	}

	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	extraction := &webExtraction{
		Engine:      "goscrapling",
		Mode:        "static",
		StatusCode:  resp.StatusCode,
		ContentType: mediaType,
		CSSSelector: selector,
	}
	if selector == "" && mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		if !webDirectTextContentAllowed(mediaType, rawURL, data) {
			return webExtractResult{}, fmt.Errorf("unsupported direct content type %q", firstNonEmpty(mediaType, http.DetectContentType(data)))
		}
		content := strings.TrimPrefix(string(data), "\ufeff")
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectTextDocumentTitle(finalURL, content),
			Content:    content,
			Extraction: extraction,
		}, nil
	}

	response, err := goscrapling.NewResponse(bytes.NewReader(data), goscrapling.ResponseOptions{
		URL:        finalURL,
		StatusCode: resp.StatusCode,
		Reason:     http.StatusText(resp.StatusCode),
		Headers:    resp.Header,
		Request: goscrapling.RequestMetadata{
			Method:  http.MethodGet,
			URL:     rawURL,
			Headers: req.Header,
		},
	})
	if err != nil {
		return webExtractResult{}, err
	}

	title := webExtractHTMLTitle(string(data))
	if selector == "" {
		contentHTML := goscraplingPreferredHTML(response)
		_, content := webExtractHTMLToMarkdown([]byte(contentHTML), finalURL)
		if strings.TrimSpace(content) == "" {
			return webExtractResult{}, fmt.Errorf("empty static HTML content")
		}
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectHTMLTitle(finalURL, title, content),
			Content:    content,
			Extraction: extraction,
		}, nil
	}

	selection := response.CSS(selector)
	if selection.Len() == 0 {
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectHTMLTitle(finalURL, title, ""),
			Error:      string(WebEvidenceInvalidArguments) + ": css_selector matched no elements",
			Evidence:   WebEvidenceInvalidArguments,
			Extraction: extraction,
		}, nil
	}

	content := webCleanMarkdownLines(selection.Text())
	if content == "" {
		if htmlContent, err := selection.HTML(); err == nil {
			content = webCleanMarkdownLines(htmlContent)
		}
	}
	if content == "" {
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectHTMLTitle(finalURL, title, ""),
			Error:      string(WebEvidenceInvalidArguments) + ": css_selector matched only empty content",
			Evidence:   WebEvidenceInvalidArguments,
			Extraction: extraction,
		}, nil
	}

	return webExtractResult{
		URL:        finalURL,
		Title:      webDirectHTMLTitle(finalURL, title, content),
		Content:    content,
		Extraction: extraction,
	}, nil
}

func (t *webTool) goscraplingBrowserExtractURL(ctx context.Context, rawURL, selector string) (webExtractResult, error) {
	selector = strings.TrimSpace(selector)
	fetcher := t.goscraplingBrowserFetcher()
	if fetcher == nil {
		return webExtractResult{}, fmt.Errorf("goscrapling browser fetcher is not configured")
	}

	reqCtx, cancel := context.WithTimeout(ctx, t.goscraplingBrowserTimeout())
	defer cancel()
	options := t.goscraplingBrowserOptions(selector)
	response, err := fetcher.Fetch(reqCtx, rawURL, options)
	if err != nil {
		return webExtractResult{}, err
	}
	extraction := t.goscraplingBrowserExtractionEvidence(rawURL, selector, response)
	finalURL := firstNonEmpty(response.URL(), rawURL)
	if result, blocked := t.blockedWebExtractRequestResult(finalURL); blocked {
		result.Extraction = extraction
		return result, nil
	}
	if status := response.StatusCode(); status < 200 || status >= 300 {
		return webExtractResult{
			URL:        finalURL,
			Error:      fmt.Sprintf("HTTP %d", status),
			Evidence:   WebEvidenceRequestFailed,
			Extraction: extraction,
		}, nil
	}

	title := webExtractHTMLTitle(response.Text())
	if selector == "" {
		contentHTML := goscraplingPreferredHTML(response)
		_, content := webExtractHTMLToMarkdown([]byte(contentHTML), finalURL)
		if strings.TrimSpace(content) == "" {
			return webExtractResult{}, fmt.Errorf("empty browser-rendered HTML content")
		}
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectHTMLTitle(finalURL, title, content),
			Content:    content,
			Extraction: extraction,
		}, nil
	}

	selection := response.CSS(selector)
	if selection.Len() == 0 {
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectHTMLTitle(finalURL, title, ""),
			Error:      string(WebEvidenceInvalidArguments) + ": css_selector matched no browser-rendered elements",
			Evidence:   WebEvidenceInvalidArguments,
			Extraction: extraction,
		}, nil
	}

	content := webCleanMarkdownLines(selection.Text())
	if content == "" {
		if htmlContent, err := selection.HTML(); err == nil {
			content = webCleanMarkdownLines(htmlContent)
		}
	}
	if content == "" {
		return webExtractResult{
			URL:        finalURL,
			Title:      webDirectHTMLTitle(finalURL, title, ""),
			Error:      string(WebEvidenceInvalidArguments) + ": css_selector matched only empty browser-rendered content",
			Evidence:   WebEvidenceInvalidArguments,
			Extraction: extraction,
		}, nil
	}

	return webExtractResult{
		URL:        finalURL,
		Title:      webDirectHTMLTitle(finalURL, title, content),
		Content:    content,
		Extraction: extraction,
	}, nil
}

func (t *webTool) goscraplingBrowserOptions(selector string) goscraplingbrowser.BrowserOptions {
	headers := http.Header{}
	headers.Set("Accept", "text/html,application/xhtml+xml,text/*;q=0.9,application/json;q=0.8,*/*;q=0.1")
	headers.Set("User-Agent", webDefaultUserAgent)
	options := goscraplingbrowser.BrowserOptions{
		Headers:          headers,
		Headless:         true,
		DisableResources: true,
		NetworkIdle:      true,
		LoadDOM:          true,
		Timeout:          t.goscraplingBrowserTimeout(),
		Wait:             t.cfg.GoscraplingBrowser.Wait,
	}
	if selector = strings.TrimSpace(selector); selector != "" {
		options.WaitSelector = goscraplingbrowser.BrowserWaitSelector{
			Selector: selector,
			State:    goscraplingbrowser.BrowserWaitVisible,
		}
	}
	return options
}

func (t *webTool) goscraplingBrowserTimeout() time.Duration {
	if t.cfg.GoscraplingBrowser.Timeout > 0 {
		return t.cfg.GoscraplingBrowser.Timeout
	}
	return t.cfg.timeout()
}

func (t *webTool) goscraplingBrowserFetcher() GoscraplingBrowserFetcher {
	if t.cfg.GoscraplingBrowser.Fetcher != nil {
		return t.cfg.GoscraplingBrowser.Fetcher
	}
	return goscraplingbrowser.BrowserFetcher{Engine: goscraplingbrowser.NewChromedpBrowserEngine(goscraplingbrowser.ChromedpBrowserOptions{})}
}

func (t *webTool) goscraplingBrowserExtractionEvidence(rawURL, selector string, response *goscrapling.Response) *webExtraction {
	statusCode := 0
	contentType := ""
	finalURL := strings.TrimSpace(rawURL)
	if response != nil {
		statusCode = response.StatusCode()
		if response.URL() != "" {
			finalURL = response.URL()
		}
		contentType, _, _ = mime.ParseMediaType(response.Headers().Get("Content-Type"))
		contentType = strings.ToLower(strings.TrimSpace(contentType))
	}
	return &webExtraction{
		Engine:       "goscrapling",
		Mode:         "browser",
		StatusCode:   statusCode,
		ContentType:  contentType,
		CSSSelector:  strings.TrimSpace(selector),
		FinalURL:     finalURL,
		WaitEvidence: goscraplingBrowserWaitEvidence(t.goscraplingBrowserOptions(selector)),
	}
}

func goscraplingBrowserWaitEvidence(options goscraplingbrowser.BrowserOptions) string {
	if options.WaitSelector.Selector != "" {
		return "selector_" + string(options.WaitSelector.State)
	}
	if options.NetworkIdle {
		return "network_idle"
	}
	if options.Wait > 0 {
		return "wait_duration"
	}
	return ""
}

func goscraplingPreferredHTML(response *goscrapling.Response) string {
	if response == nil {
		return ""
	}
	for _, selector := range []string{"main", "article", "body"} {
		selection := response.CSS(selector)
		if selection.Len() == 0 {
			continue
		}
		body, err := selection.HTML()
		if err == nil && strings.TrimSpace(body) != "" {
			return body
		}
	}
	return response.Text()
}

func (t *webTool) executeExtractBackend(ctx context.Context, urls []string) (map[string]any, error) {
	switch t.cfg.Resolution.Backend {
	case WebBackendTavily:
		return t.postProviderJSON(ctx, "extract", map[string]any{
			"urls":           urls,
			"include_images": false,
			"api_key":        t.cfg.Resolution.APIKey,
		}, nil)
	case WebBackendExa:
		return t.postProviderJSON(ctx, "contents", map[string]any{
			"urls": urls,
			"text": true,
		}, map[string]string{
			"x-api-key":         t.cfg.Resolution.APIKey,
			"x-exa-integration": "hermes-agent",
		})
	case WebBackendParallel:
		return t.postProviderJSON(ctx, "v1beta/extract", map[string]any{
			"urls":         urls,
			"full_content": true,
		}, map[string]string{
			"x-api-key": t.cfg.Resolution.APIKey,
		})
	case WebBackendDuckDuckGo:
		return t.duckDuckGoExtract(ctx, urls)
	default:
		return nil, fmt.Errorf("web: unsupported extract backend %q", t.cfg.Resolution.Backend)
	}
}

func (t *webTool) executeCrawl(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		URL          string `json:"url"`
		Instructions string `json:"instructions"`
		Depth        string `json:"depth"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("%s: invalid args: %w", t.name, err)
	}
	crawlURL := normalizeWebCrawlURL(in.URL)
	if crawlURL == "" {
		return nil, fmt.Errorf("%s: url is required", t.name)
	}
	if webURLContainsSecret(crawlURL) {
		return json.Marshal(webErrorResponse{
			Success: false,
			Error:   "Blocked: URL contains what appears to be an API key or token. Secrets must not be sent in URLs.",
		})
	}

	if t.cfg.Resolution.Backend == WebBackendTavily && t.cfg.Resolution.Available {
		return t.executeTavilyCrawl(ctx, crawlURL, in.Instructions, in.Depth)
	}

	if t.cfg.Resolution.Backend == WebBackendGoscraplingCrawler {
		return t.executeGoscraplingCrawlerCrawl(ctx, crawlURL, in.Instructions, in.Depth)
	}

	if t.cfg.Resolution.Backend != WebBackendFirecrawl || !t.cfg.Resolution.Available {
		return json.Marshal(webErrorResponse{
			Success:  false,
			Error:    "web_crawl requires Firecrawl or Tavily. Set FIRECRAWL_API_KEY, FIRECRAWL_API_URL, TAVILY_API_KEY, or use web_search + web_extract instead.",
			Evidence: WebEvidenceProviderUnavailable,
			Backend:  t.cfg.Resolution.Backend,
		})
	}

	if errResult, blocked := t.blockedWebExtractRequestResult(crawlURL); blocked {
		return marshalWebExtractResponse(webExtractResponse{Results: []webExtractResult{errResult}})
	}

	raw, err := t.postFirecrawlJSON(ctx, "crawl", map[string]any{
		"url":   crawlURL,
		"limit": defaultWebCrawlLimit,
		"scrape_options": map[string]any{
			"formats": []string{"markdown"},
		},
	})
	if err != nil {
		return json.Marshal(webErrorResponse{
			Success: false,
			Error:   "Error crawling website: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
		})
	}
	return t.marshalProcessedCrawlResults(ctx, normalizeWebCrawlDocuments(crawlURL, raw))
}

func (t *webTool) executeGoscraplingCrawlerCrawl(ctx context.Context, crawlURL, instructions, depth string) (json.RawMessage, error) {
	if errResult, blocked := t.blockedWebExtractRequestResult(crawlURL); blocked {
		return marshalWebExtractResponse(webExtractResponse{Results: []webExtractResult{errResult}})
	}
	crawler := t.cfg.GoscraplingCrawler.Crawler
	if crawler == nil {
		return json.Marshal(goscraplingCrawlerUnavailableResponse("local goscrapling crawler adapter not yet available"))
	}
	maxPages := t.cfg.GoscraplingCrawler.MaxPages
	if maxPages <= 0 {
		maxPages = defaultWebCrawlLimit
	}
	result, err := crawler.Crawl(ctx, GoscraplingCrawlRequest{URL: crawlURL, Instructions: strings.TrimSpace(instructions), Depth: strings.TrimSpace(depth), MaxPages: maxPages})
	if err != nil {
		return json.Marshal(webErrorResponse{
			Success:        false,
			Error:          "Error crawling website with local goscrapling crawler: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
			Evidence:       WebEvidenceRequestFailed,
			Backend:        WebBackendGoscraplingCrawler,
			Degraded:       true,
			DegradedReason: "local goscrapling crawler adapter failed",
		})
	}
	return t.marshalProcessedCrawlResults(ctx, normalizeGoscraplingCrawlPages(crawlURL, result, maxPages))
}

func goscraplingCrawlerUnavailableResponse(reason string) webErrorResponse {
	return webErrorResponse{
		Success:        false,
		Error:          "local goscrapling crawler backend is not yet available. Use Firecrawl or Tavily for web_crawl until the local crawler adapter gate is complete.",
		Evidence:       WebEvidenceProviderUnavailable,
		Backend:        WebBackendGoscraplingCrawler,
		Degraded:       true,
		DegradedReason: reason,
	}
}

func normalizeGoscraplingCrawlPages(requestedURL string, result GoscraplingCrawlResult, maxPages int) []webExtractResult {
	stats := result.Stats
	if stats.MaxPages == 0 {
		stats.MaxPages = maxPages
	}
	out := make([]webExtractResult, 0, len(result.Pages))
	for _, page := range result.Pages {
		if page.Duplicate || page.Offsite {
			continue
		}
		pageURL := firstNonEmpty(page.URL, page.FinalURL, requestedURL)
		finalURL := firstNonEmpty(page.FinalURL, pageURL)
		item := webExtractResult{
			URL:     pageURL,
			Title:   page.Title,
			Content: page.Content,
			Extraction: &webExtraction{
				Engine:      "goscrapling",
				Mode:        "crawler",
				StatusCode:  page.StatusCode,
				ContentType: page.ContentType,
				FinalURL:    finalURL,
				Crawl:       &stats,
			},
		}
		if strings.TrimSpace(page.Error) != "" {
			item.Error = page.Error
			item.Evidence = page.Evidence
			if item.Evidence == "" {
				item.Evidence = WebEvidenceRequestFailed
			}
		}
		out = append(out, item)
	}
	return out
}

func (t *webTool) executeTavilyCrawl(ctx context.Context, crawlURL, instructions, depth string) (json.RawMessage, error) {
	if errResult, blocked := t.blockedWebExtractRequestResult(crawlURL); blocked {
		return marshalWebExtractResponse(webExtractResponse{Results: []webExtractResult{errResult}})
	}
	payload := map[string]any{
		"url":           crawlURL,
		"limit":         defaultWebCrawlLimit,
		"extract_depth": webCrawlDepth(depth),
		"api_key":       t.cfg.Resolution.APIKey,
	}
	if strings.TrimSpace(instructions) != "" {
		payload["instructions"] = instructions
	}
	raw, err := t.postProviderJSON(ctx, "crawl", payload, nil)
	if err != nil {
		return json.Marshal(webErrorResponse{
			Success: false,
			Error:   "Error crawling website: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
		})
	}
	return t.marshalProcessedCrawlResults(ctx, normalizeWebExtractDocuments([]string{crawlURL}, "markdown", raw))
}

func (t *webTool) marshalProcessedCrawlResults(ctx context.Context, results []webExtractResult) (json.RawMessage, error) {
	out := make([]webExtractResult, 0, len(results))
	for _, result := range results {
		out = append(out, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result)))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: out})
}

func (t *webTool) postFirecrawlJSON(ctx context.Context, endpoint string, payload any) (map[string]any, error) {
	u, err := webFirecrawlEndpoint(t.cfg.Resolution.BaseURL, endpoint)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	if t.cfg.Resolution.APIKey != "" {
		headers["Authorization"] = "Bearer " + t.cfg.Resolution.APIKey
	}
	return t.postJSON(ctx, u, payload, headers)
}

func (t *webTool) postProviderJSON(ctx context.Context, endpoint string, payload any, headers map[string]string) (map[string]any, error) {
	u, err := webProviderEndpoint(t.cfg.Resolution.BaseURL, endpoint)
	if err != nil {
		return nil, err
	}
	return t.postJSON(ctx, u, payload, headers)
}

func (t *webTool) getJSON(ctx context.Context, endpointURL string, headers map[string]string) (map[string]any, error) {
	data, err := t.getRaw(ctx, endpointURL, headers)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("web: decode response: %w", err)
	}
	return out, nil
}

func (t *webTool) getRaw(ctx context.Context, endpointURL string, headers map[string]string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, t.cfg.timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("web: build request: %w", err)
	}
	req.Header.Set("Accept", "*/*")
	for k, v := range headers {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := t.cfg.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("web: do request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxWebResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("web: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("web: unexpected status %d: %s", resp.StatusCode, safeWebSnippet(data))
	}
	return data, nil
}

func (t *webTool) postJSON(ctx context.Context, endpointURL string, payload any, headers map[string]string) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("web: marshal payload: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, t.cfg.timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpointURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("web: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := t.cfg.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("web: do request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxWebResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("web: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("web: unexpected status %d: %s", resp.StatusCode, safeWebSnippet(data))
	}
	var out map[string]any
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("web: decode response: %w", err)
	}
	return out, nil
}

func (t *webTool) duckDuckGoExtract(ctx context.Context, urls []string) (map[string]any, error) {
	results := make([]map[string]any, 0, len(urls))
	pendingInstantAnswer := make([]string, 0, len(urls))
	for _, rawURL := range urls {
		result, err := t.goscraplingExtractURL(ctx, rawURL, "")
		if err == nil {
			results = append(results, webExtractResultProviderRow(result))
			continue
		}
		pendingInstantAnswer = append(pendingInstantAnswer, rawURL)
	}
	if len(pendingInstantAnswer) == 0 {
		return map[string]any{"results": results}, nil
	}

	raw, err := t.getRaw(ctx, fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", url.QueryEscape(strings.Join(pendingInstantAnswer, " "))), map[string]string{
		"Accept":     "application/json",
		"User-Agent": webDefaultUserAgent,
	})
	if err != nil {
		return nil, err
	}
	var ddg struct {
		AbstractText   string `json:"AbstractText"`
		AbstractSource string `json:"AbstractSource"`
		AbstractURL    string `json:"AbstractURL"`
		Heading        string `json:"Heading"`
		Answer         string `json:"Answer"`
	}
	if err := json.Unmarshal(raw, &ddg); err != nil {
		return nil, fmt.Errorf("web: invalid ddg response: %w", err)
	}
	if ddg.AbstractText == "" && ddg.Answer == "" {
		results = append(results, map[string]any{
			"title":   "DuckDuckGo Instant Answer",
			"content": "No instant answer available for this query.",
			"url":     strings.Join(pendingInstantAnswer, " "),
		})
		return map[string]any{"results": results}, nil
	}
	results = append(results, map[string]any{
		"title":   firstNonEmpty(ddg.Heading, ddg.AbstractSource, "DuckDuckGo Instant Answer"),
		"content": firstNonEmpty(ddg.Answer, ddg.AbstractText),
		"url":     firstNonEmpty(ddg.AbstractURL, strings.Join(pendingInstantAnswer, " ")),
	})
	return map[string]any{"results": results}, nil
}

func (t *webTool) directTextDocumentHTTPClient() WebHTTPClient {
	if t.cfg.Client != nil {
		return t.cfg.Client
	}
	return &http.Client{
		Timeout: t.cfg.timeout(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("web: too many redirects")
			}
			if _, blocked := t.blockedWebExtractRequestResult(req.URL.String()); blocked {
				return fmt.Errorf("web: redirected to blocked URL")
			}
			return nil
		},
	}
}

func webLikelyDirectTextDocumentURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".txt", ".md", ".markdown", ".mdx", ".rst", ".adoc", ".csv", ".tsv", ".json", ".jsonl", ".yaml", ".yml", ".toml", ".xml", ".rss", ".atom", ".log":
		return true
	default:
		return false
	}
}

func webExtractHTMLToMarkdown(data []byte, rawURL string) (string, string) {
	htmlText := string(data)
	title := webExtractHTMLTitle(htmlText)
	htmlText = webPreferredHTMLContent(htmlText)
	htmlText = webStripHTMLNoise(htmlText)
	htmlText = webRenderHTMLLinks(htmlText, rawURL)
	htmlText = webRenderHTMLBlocks(htmlText)
	htmlText = webHTMLTagPattern.ReplaceAllString(htmlText, "")
	htmlText = html.UnescapeString(htmlText)
	content := webCleanMarkdownLines(htmlText)
	return title, content
}

func webExtractHTMLTitle(htmlText string) string {
	match := regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`).FindStringSubmatch(htmlText)
	if len(match) < 2 {
		return ""
	}
	return webCleanHTMLInline(match[1])
}

func webPreferredHTMLContent(htmlText string) string {
	for _, tag := range []string{"main", "article", "body"} {
		re := regexp.MustCompile(`(?is)<` + tag + `\b[^>]*>(.*?)</` + tag + `>`)
		if match := re.FindStringSubmatch(htmlText); len(match) >= 2 {
			return match[1]
		}
	}
	return htmlText
}

func webStripHTMLNoise(htmlText string) string {
	for _, tag := range []string{"script", "style", "svg", "canvas", "noscript", "template", "nav", "header", "footer", "aside", "form", "button"} {
		re := regexp.MustCompile(`(?is)<` + tag + `\b[^>]*>.*?</` + tag + `>`)
		htmlText = re.ReplaceAllString(htmlText, "")
	}
	return htmlText
}

func webRenderHTMLLinks(htmlText, baseURL string) string {
	re := regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
	return re.ReplaceAllStringFunc(htmlText, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 3 {
			return ""
		}
		label := webCleanHTMLInline(parts[2])
		if label == "" {
			return ""
		}
		href := webResolveHTMLHref(baseURL, webHTMLAttr(parts[1], "href"))
		if href == "" {
			return label
		}
		return label + " (" + href + ")"
	})
}

func webRenderHTMLBlocks(htmlText string) string {
	for level := 6; level >= 1; level-- {
		re := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d\b[^>]*>(.*?)</h%d>`, level, level))
		prefix := strings.Repeat("#", level)
		htmlText = re.ReplaceAllStringFunc(htmlText, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) < 2 {
				return ""
			}
			heading := webCleanHTMLInline(parts[1])
			if heading == "" {
				return ""
			}
			return "\n\n" + prefix + " " + heading + "\n\n"
		})
	}
	preRe := regexp.MustCompile(`(?is)<pre\b[^>]*>(.*?)</pre>`)
	htmlText = preRe.ReplaceAllStringFunc(htmlText, func(match string) string {
		parts := preRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return ""
		}
		code := strings.TrimSpace(html.UnescapeString(webHTMLTagPattern.ReplaceAllString(parts[1], "")))
		if code == "" {
			return ""
		}
		return "\n\n```\n" + code + "\n```\n\n"
	})
	liRe := regexp.MustCompile(`(?is)<li\b[^>]*>(.*?)</li>`)
	htmlText = liRe.ReplaceAllStringFunc(htmlText, func(match string) string {
		parts := liRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return ""
		}
		item := webCleanHTMLInline(parts[1])
		if item == "" {
			return ""
		}
		return "\n- " + item + "\n"
	})
	htmlText = regexp.MustCompile(`(?is)<br\s*/?>`).ReplaceAllString(htmlText, "\n")
	htmlText = regexp.MustCompile(`(?is)</?(p|div|section|ul|ol|table|tr|blockquote)\b[^>]*>`).ReplaceAllString(htmlText, "\n\n")
	return htmlText
}

func webHTMLAttr(attrs, key string) string {
	re := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(key) + `\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	match := re.FindStringSubmatch(attrs)
	if len(match) == 0 {
		return ""
	}
	for _, value := range match[2:] {
		if value != "" {
			return html.UnescapeString(strings.TrimSpace(value))
		}
	}
	return ""
}

func webResolveHTMLHref(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(strings.ToLower(href), "javascript:") {
		return ""
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			return parsed.String()
		}
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

func webCleanHTMLInline(raw string) string {
	text := webHTMLTagPattern.ReplaceAllString(raw, "")
	text = html.UnescapeString(text)
	return strings.Join(strings.Fields(text), " ")
}

func webCleanMarkdownLines(raw string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !blank && len(out) > 0 {
				out = append(out, "")
				blank = true
			}
			continue
		}
		trimmed = strings.Join(strings.Fields(trimmed), " ")
		out = append(out, trimmed)
		blank = false
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func webDirectHTMLTitle(rawURL, title, content string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	return webDirectTextDocumentTitle(rawURL, content)
}

func webDirectTextContentAllowed(mediaType, rawURL string, data []byte) bool {
	switch mediaType {
	case "text/html", "application/xhtml+xml":
		return false
	case "text/plain", "text/markdown", "text/x-markdown", "text/csv", "text/tab-separated-values", "text/xml",
		"application/markdown", "application/x-markdown", "application/json", "application/ld+json", "application/xml",
		"application/yaml", "application/x-yaml", "text/yaml", "application/toml":
		return true
	}
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	if mediaType != "" && mediaType != "application/octet-stream" {
		return false
	}
	if !webLikelyDirectTextDocumentURL(rawURL) {
		return false
	}
	if bytes.Contains(data, []byte{0}) {
		return false
	}
	sniffed := strings.ToLower(http.DetectContentType(data))
	return strings.HasPrefix(sniffed, "text/plain") || strings.Contains(sniffed, "json") || strings.Contains(sniffed, "xml")
}

func webDirectTextDocumentTitle(rawURL, content string) string {
	if parsed, err := url.Parse(rawURL); err == nil {
		if base := path.Base(parsed.Path); base != "." && base != "/" && base != "" {
			return base
		}
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if trimmed != "" {
			return trimmed
		}
	}
	return rawURL
}

func webExtractResultProviderRow(result webExtractResult) map[string]any {
	row := map[string]any{
		"url":     result.URL,
		"title":   result.Title,
		"content": result.Content,
	}
	if result.Error != "" {
		row["error"] = result.Error
	}
	if result.Extraction != nil {
		row["extraction"] = result.Extraction
	}
	return row
}

func webExtractionFromRaw(raw any) *webExtraction {
	switch value := raw.(type) {
	case *webExtraction:
		if value == nil {
			return nil
		}
		clone := *value
		if webExtractionEmpty(clone) {
			return nil
		}
		return &clone
	case webExtraction:
		if webExtractionEmpty(value) {
			return nil
		}
		clone := value
		return &clone
	case map[string]any:
		extraction := webExtraction{
			Engine:       webStringValue(value["engine"]),
			Mode:         webStringValue(value["mode"]),
			StatusCode:   intValue(value["status_code"]),
			ContentType:  webStringValue(value["content_type"]),
			CSSSelector:  webStringValue(value["css_selector"]),
			FinalURL:     webStringValue(value["final_url"]),
			WaitEvidence: webStringValue(value["wait_evidence"]),
		}
		if webExtractionEmpty(extraction) {
			return nil
		}
		return &extraction
	default:
		return nil
	}
}

func webExtractionEmpty(extraction webExtraction) bool {
	return extraction.Engine == "" &&
		extraction.Mode == "" &&
		extraction.StatusCode == 0 &&
		extraction.ContentType == "" &&
		extraction.CSSSelector == "" &&
		extraction.FinalURL == "" &&
		extraction.WaitEvidence == ""
}

var (
	webDuckDuckGoLinkPattern    = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*\bresult__a\b[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	webDuckDuckGoSnippetPattern = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*\bresult__snippet\b[^"]*"[^>]*>(.*?)</a>`)
	webHTMLTagPattern           = regexp.MustCompile(`(?is)<[^>]+>`)
)

func duckDuckGoSearchResults(body []byte, limit int) []any {
	if limit <= 0 {
		limit = defaultWebSearchLimit
	}
	htmlText := string(body)
	links := webDuckDuckGoLinkPattern.FindAllStringSubmatch(htmlText, limit+5)
	snippets := webDuckDuckGoSnippetPattern.FindAllStringSubmatch(htmlText, limit+5)
	results := make([]any, 0, minInt(len(links), limit))
	for i, match := range links {
		if len(results) >= limit || len(match) < 3 {
			break
		}
		title := cleanDuckDuckGoHTMLText(match[2])
		resultURL := decodeDuckDuckGoURL(html.UnescapeString(match[1]))
		if title == "" || resultURL == "" {
			continue
		}
		row := map[string]any{
			"title":    title,
			"url":      resultURL,
			"position": len(results) + 1,
		}
		if i < len(snippets) && len(snippets[i]) >= 2 {
			if snippet := cleanDuckDuckGoHTMLText(snippets[i][1]); snippet != "" {
				row["snippet"] = snippet
			}
		}
		results = append(results, row)
	}
	return results
}

// duckDuckGoLiteSearchResults parses the simpler HTML from lite.duckduckgo.com.
// Lite uses a table layout with plain <a> links, which is more stable than the
// CSS class-based parsing on the main HTML endpoint.
func duckDuckGoLiteSearchResults(body []byte, limit int) []any {
	if limit <= 0 {
		limit = defaultWebSearchLimit
	}
	htmlText := string(body)
	linkPattern := regexp.MustCompile(`(?is)<a[^>]*href="\/\/?([^"]+)"[^>]*class="[^"]*result-link[^"]*"[^>]*>(.*?)</a>`)
	snippetPattern := regexp.MustCompile(`(?is)<td[^>]*class="[^"]*result-snippet[^"]*"[^>]*>(.*?)</td>`)
	links := linkPattern.FindAllStringSubmatch(htmlText, limit+5)
	snippets := snippetPattern.FindAllStringSubmatch(htmlText, limit+5)
	results := make([]any, 0, minInt(len(links), limit))
	for i, match := range links {
		if len(results) >= limit || len(match) < 3 {
			break
		}
		title := cleanDuckDuckGoHTMLText(match[2])
		resultURL := decodeDuckDuckGoURL("https://" + match[1])
		if title == "" || resultURL == "" {
			continue
		}
		row := map[string]any{
			"title":    title,
			"url":      resultURL,
			"position": len(results) + 1,
		}
		if i < len(snippets) && len(snippets[i]) >= 2 {
			if snippet := cleanDuckDuckGoHTMLText(snippets[i][1]); snippet != "" {
				row["snippet"] = snippet
			}
		}
		results = append(results, row)
	}
	return results
}

func decodeDuckDuckGoURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil {
		if redirected := parsed.Query().Get("uddg"); redirected != "" {
			return strings.TrimSpace(redirected)
		}
	}
	if decoded, err := url.QueryUnescape(value); err == nil && decoded != value {
		if parsed, err := url.Parse(decoded); err == nil {
			if redirected := parsed.Query().Get("uddg"); redirected != "" {
				return strings.TrimSpace(redirected)
			}
		}
		value = decoded
	}
	return strings.TrimSpace(value)
}

func cleanDuckDuckGoHTMLText(raw string) string {
	text := webHTMLTagPattern.ReplaceAllString(raw, "")
	text = html.UnescapeString(text)
	return strings.Join(strings.Fields(text), " ")
}

func mapList(v any) []map[string]any {
	if rows, ok := v.([]map[string]any); ok {
		return rows
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

func firstRequestedURL(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func webExtractContent(payload map[string]any, format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "html":
		return firstNonEmpty(webStringValue(payload["html"]), webStringValue(payload["markdown"]), webStringValue(payload["content"]))
	default:
		return firstNonEmpty(webStringValue(payload["markdown"]), webStringValue(payload["content"]), webStringValue(payload["html"]))
	}
}

func blockedWebExtractResult(rawURL string) (webExtractResult, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return webExtractResult{
			URL:      rawURL,
			Error:    string(WebEvidenceInvalidArguments) + ": url must be absolute http(s)",
			Evidence: WebEvidenceInvalidArguments,
		}, true
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return webExtractResult{
			URL:      rawURL,
			Error:    string(WebEvidenceInvalidArguments) + ": url scheme must be http or https",
			Evidence: WebEvidenceInvalidArguments,
		}, true
	}
	if IsPrivateBrowserHost(parsed.Hostname()) {
		return webExtractResult{
			URL:      rawURL,
			Error:    string(WebEvidencePrivateURLBlocked) + ": URL targets a private or internal network address",
			Evidence: WebEvidencePrivateURLBlocked,
		}, true
	}
	return webExtractResult{}, false
}

func (t *webTool) blockedWebExtractRequestResult(rawURL string) (webExtractResult, bool) {
	if result, blocked := blockedWebExtractResult(rawURL); blocked {
		return result, true
	}
	if block, blocked := t.cfg.Policy.Check(rawURL); blocked {
		return webPolicyBlockedResult(rawURL, "", block), true
	}
	return webExtractResult{}, false
}

func (t *webTool) applyWebExtractPostPolicy(result webExtractResult) webExtractResult {
	if result.URL == "" {
		return result
	}
	if block, blocked := t.cfg.Policy.Check(result.URL); blocked {
		return webPolicyBlockedResult(result.URL, result.Title, block)
	}
	return result
}

func (t *webTool) processWebExtractResult(ctx context.Context, result webExtractResult) webExtractResult {
	if !t.cfg.Processing.Enabled || t.cfg.ContentProcessor == nil || result.Error != "" {
		return result
	}
	if len(result.Content) == 0 {
		return result
	}
	maxInput := t.cfg.processMaxInput()
	maxOutput := t.cfg.processMaxOutput()
	if maxInput > 0 && len(result.Content) > maxInput {
		sizeMB := float64(len(result.Content)) / 1_000_000
		result.Content = fmt.Sprintf("[Content too large to process: %.1fMB. Try using web_crawl with specific extraction instructions, or search for a more focused source.]", sizeMB)
		return result
	}
	if len(result.Content) < t.cfg.processMinLength() {
		return result
	}

	processed, err := t.cfg.ContentProcessor.ProcessWebContent(ctx, WebContentProcessRequest{
		URL:     result.URL,
		Title:   result.Title,
		Content: result.Content,
	})
	if err != nil || strings.TrimSpace(processed) == "" {
		result.Content = webTruncateContent(result.Content, maxOutput, "\n\n[Content truncated - LLM summarization failed. Use browser_navigate for the full page.]")
		return result
	}
	result.Content = webTruncateContent(processed, maxOutput, "\n\n[... summary truncated for context management ...]")
	return result
}

func webTruncateContent(content string, maxChars int, suffix string) string {
	if maxChars <= 0 || len(content) <= maxChars {
		return content
	}
	return content[:maxChars] + suffix
}

type webWebsitePolicyBlock struct {
	Host    string
	Rule    string
	Source  string
	Message string
}

func (p WebWebsitePolicy) Check(rawURL string) (webWebsitePolicyBlock, bool) {
	if !p.Enabled {
		return webWebsitePolicyBlock{}, false
	}
	host := webPolicyHost(rawURL)
	if host == "" {
		return webWebsitePolicyBlock{}, false
	}
	for _, rule := range p.rules() {
		if webPolicyHostMatches(host, rule.pattern) {
			return webWebsitePolicyBlock{
				Host:    host,
				Rule:    rule.pattern,
				Source:  rule.source,
				Message: fmt.Sprintf("Blocked by website policy: '%s' matched rule '%s' from %s", host, rule.pattern, rule.source),
			}, true
		}
	}
	return webWebsitePolicyBlock{}, false
}

type webWebsitePolicyRule struct {
	pattern string
	source  string
}

func (p WebWebsitePolicy) rules() []webWebsitePolicyRule {
	rules := make([]webWebsitePolicyRule, 0, len(p.Domains))
	seen := map[string]struct{}{}
	add := func(source, raw string) {
		normalized := webPolicyNormalizeRule(raw)
		if normalized == "" {
			return
		}
		key := source + "\x00" + normalized
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		rules = append(rules, webWebsitePolicyRule{pattern: normalized, source: source})
	}
	for _, raw := range p.Domains {
		add("config", raw)
	}
	for _, sharedFile := range p.SharedFiles {
		pathValue := strings.TrimSpace(sharedFile)
		if pathValue == "" {
			continue
		}
		if !path.IsAbs(pathValue) && strings.TrimSpace(p.SharedFileBaseDir) != "" {
			pathValue = path.Join(p.SharedFileBaseDir, pathValue)
		}
		body, err := os.ReadFile(pathValue)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(body), "\n") {
			add(pathValue, line)
		}
	}
	return rules
}

func webPolicyBlockedResult(rawURL, title string, block webWebsitePolicyBlock) webExtractResult {
	return webExtractResult{
		URL:      rawURL,
		Title:    title,
		Content:  "",
		Error:    block.Message,
		Evidence: WebEvidenceWebsitePolicy,
		BlockedByPolicy: map[string]string{
			"host":   block.Host,
			"rule":   block.Rule,
			"source": block.Source,
		},
	}
}

func webPolicyHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	host := ""
	if err == nil {
		host = parsed.Hostname()
		if host == "" {
			host = parsed.Host
		}
	}
	if host == "" && !strings.Contains(rawURL, "://") {
		if schemeless, err := url.Parse("//" + rawURL); err == nil {
			host = schemeless.Hostname()
			if host == "" {
				host = schemeless.Host
			}
		}
	}
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.TrimSuffix(host, ".")
}

func webPolicyNormalizeRule(raw any) string {
	rule, ok := raw.(string)
	if !ok {
		return ""
	}
	value := strings.ToLower(strings.TrimSpace(rule))
	if value == "" || strings.HasPrefix(value, "#") {
		return ""
	}
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil {
			if parsed.Host != "" {
				value = parsed.Host
			} else {
				value = parsed.Path
			}
		}
	}
	value = strings.SplitN(value, "/", 2)[0]
	value = strings.TrimSuffix(strings.TrimSpace(value), ".")
	if strings.HasPrefix(value, "www.") {
		value = strings.TrimPrefix(value, "www.")
	}
	return value
}

func webPolicyHostMatches(host, pattern string) bool {
	if host == "" || pattern == "" {
		return false
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(pattern, "*.")
	}
	return host == pattern || strings.HasSuffix(host, "."+pattern)
}

func normalizeWebCrawlURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "https://" + value
	}
	return value
}

func webCrawlDepth(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "advanced":
		return "advanced"
	default:
		return "basic"
	}
}

func webRequiresEnv(includeManaged bool) []string {
	requires := []string{
		"EXA_API_KEY",
		"PARALLEL_API_KEY",
		"TAVILY_API_KEY",
		"FIRECRAWL_API_KEY",
		"FIRECRAWL_API_URL",
		"BRAVE_API_KEY",
		"SEARXNG_BASE_URL",
		"PERPLEXITY_API_KEY",
		"CHROME_REMOTE_DEBUGGING_URL",
		"BROWSER_CDP_URL",
	}
	if includeManaged {
		requires = append(requires,
			"FIRECRAWL_GATEWAY_URL",
			"TOOL_GATEWAY_DOMAIN",
			"TOOL_GATEWAY_SCHEME",
			"TOOL_GATEWAY_USER_TOKEN",
		)
	}
	return requires
}

func webBackendToolNames(backend WebBackend) []string {
	if backend == WebBackendCDP || backend == WebBackendGoscraplingBrowser {
		return []string{WebToolExtract}
	}
	if backend == WebBackendGoscraplingCrawler {
		return []string{WebToolCrawl}
	}
	if backend == WebBackendDuckDuckGo {
		return []string{WebToolSearch, WebToolExtract}
	}
	if webBackendSearchOnly(backend) {
		return []string{WebToolSearch}
	}
	return []string{WebToolSearch, WebToolExtract, WebToolCrawl}
}

func webBackendSearchOnly(backend WebBackend) bool {
	switch backend {
	case WebBackendBrave, WebBackendSearXNG, WebBackendPerplexity:
		return true
	default:
		return false
	}
}

func webProviderUnavailableMessage() string {
	return "Web tools are not configured. Set FIRECRAWL_API_KEY, FIRECRAWL_API_URL, PARALLEL_API_KEY, TAVILY_API_KEY, EXA_API_KEY, BRAVE_API_KEY, SEARXNG_BASE_URL, PERPLEXITY_API_KEY, or CHROME_REMOTE_DEBUGGING_URL for browser-backed web_extract. DuckDuckGo search and goscrapling static extraction fallback are automatic when no API keys are configured."
}

func webExtractFormats(format string) []string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "markdown":
		return []string{"markdown"}
	case "html":
		return []string{"html"}
	default:
		return []string{"markdown", "html"}
	}
}

func marshalWebExtractResponse(resp webExtractResponse) (json.RawMessage, error) {
	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(cleanWebBase64Images(string(raw))), nil
}

var (
	webBase64ImagePattern = regexp.MustCompile(`\(?data:image/[^;()\s]+;base64,[A-Za-z0-9+/=]+\)?`)
)

func cleanWebBase64Images(text string) string {
	return webBase64ImagePattern.ReplaceAllStringFunc(text, func(match string) string {
		if strings.HasPrefix(match, "(") && strings.HasSuffix(match, ")") {
			return "[BASE64_IMAGE_REMOVED]"
		}
		return "[BASE64_IMAGE_REMOVED]"
	})
}

var webSecretPrefixPattern = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_-])((sk|tvly|exa|fc|pplx|fal|hf|r8|npm|gsk|hsk|brv|mem0)[-_][A-Za-z0-9_-]{10,}|(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{10,}|github_pat_[A-Za-z0-9_]{10,}|xox[baprs]-[A-Za-z0-9-]{10,}|AIza[A-Za-z0-9_-]{30,}|AKIA[A-Z0-9]{16}|SG\.[A-Za-z0-9_-]{10,})([^A-Za-z0-9_-]|$)`)

func webURLsContainSecrets(urls []string) bool {
	for _, rawURL := range urls {
		if webURLContainsSecret(rawURL) {
			return true
		}
	}
	return false
}

func webURLContainsSecret(rawURL string) bool {
	candidates := []string{rawURL}
	if decoded, err := url.QueryUnescape(rawURL); err == nil && decoded != rawURL {
		candidates = append(candidates, decoded)
	}
	if decoded, err := url.PathUnescape(rawURL); err == nil && decoded != rawURL {
		candidates = append(candidates, decoded)
	}
	for _, candidate := range candidates {
		if webSecretPrefixPattern.MatchString(candidate) {
			return true
		}
	}
	return false
}

func webSearchFailure(message string, evidence WebEvidence) (json.RawMessage, error) {
	return json.Marshal(webSearchResponse{
		Success:  false,
		Error:    message,
		Evidence: evidence,
	})
}

func webFirecrawlEndpoint(baseURL, endpoint string) (string, error) {
	return webEndpoint(baseURL, "v2", endpoint)
}

func webProviderEndpoint(baseURL, endpoint string) (string, error) {
	return webEndpoint(baseURL, endpoint)
}

func webEndpoint(baseURL string, elements ...string) (string, error) {
	normalized := normalizeWebBaseURL(baseURL)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("web: invalid base url %q", baseURL)
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	parts := append([]string{basePath}, elements...)
	parsed.Path = path.Join(parts...)
	if !strings.HasPrefix(parsed.Path, "/") {
		parsed.Path = "/" + parsed.Path
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func webURLWithQuery(endpointURL string, values url.Values) (string, error) {
	parsed, err := url.Parse(endpointURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("web: invalid endpoint url %q", endpointURL)
	}
	q := parsed.Query()
	for key, vals := range values {
		q.Del(key)
		for _, value := range vals {
			q.Add(key, value)
		}
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func normalizeWebBaseURL(raw string) string {
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	if value == "" {
		value = webDefaultFirecrawlBaseURL
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	cleanPath := strings.TrimRight(parsed.Path, "/")
	if cleanPath == "/v1" || cleanPath == "/v2" {
		cleanPath = ""
	}
	parsed.Path = cleanPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func managedFirecrawlGatewayURL(domain, scheme string) string {
	domain = strings.TrimSpace(domain)
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if scheme == "" {
		scheme = "https"
	}
	if strings.Contains(domain, "://") {
		return domain
	}
	if domain == "" {
		domain = "nousresearch.com"
	}
	return scheme + "://firecrawl-gateway." + strings.TrimPrefix(domain, ".")
}

func validWebGatewayURLScheme(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return true
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

type webManagedAuthStore struct {
	Providers map[string]map[string]any `json:"providers"`
}

func readNousAccessTokenFromAuthStore(authStorePath string, now time.Time, client WebHTTPClient, timeout time.Duration) (string, bool) {
	pathValue := strings.TrimSpace(authStorePath)
	if pathValue == "" {
		return "", false
	}
	raw, err := os.ReadFile(pathValue)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return "", false
	}
	var store webManagedAuthStore
	if err := json.Unmarshal(raw, &store); err != nil {
		return "", false
	}
	provider := store.Providers["nous"]
	token := strings.TrimSpace(webStringValue(provider["access_token"]))
	if token == "" {
		return "", false
	}
	if !webAuthTokenIsExpiring(provider["expires_at"], now) {
		return token, true
	}
	refreshed, ok := refreshNousAccessToken(provider, client, timeout)
	if !ok {
		return token, true
	}
	for key, value := range refreshed {
		provider[key] = value
	}
	provider["obtained_at"] = now.UTC().Format(time.RFC3339)
	if expiresIn := intValue(refreshed["expires_in"]); expiresIn > 0 {
		provider["expires_at"] = now.UTC().Add(time.Duration(expiresIn) * time.Second).Format(time.RFC3339)
	}
	if err := writeWebManagedAuthStore(pathValue, raw, store); err != nil {
		return token, true
	}
	return strings.TrimSpace(webStringValue(provider["access_token"])), true
}

func refreshNousAccessToken(provider map[string]any, client WebHTTPClient, timeout time.Duration) (map[string]any, bool) {
	refreshToken := strings.TrimSpace(webStringValue(provider["refresh_token"]))
	if refreshToken == "" || client == nil {
		return nil, false
	}
	portalBaseURL := firstNonEmpty(
		strings.TrimRight(strings.TrimSpace(webStringValue(provider["portal_base_url"])), "/"),
		strings.TrimRight(strings.TrimSpace(os.Getenv("HERMES_PORTAL_BASE_URL")), "/"),
		strings.TrimRight(strings.TrimSpace(os.Getenv("NOUS_PORTAL_BASE_URL")), "/"),
		"https://portal.nousresearch.com",
	)
	clientID := firstNonEmpty(strings.TrimSpace(webStringValue(provider["client_id"])), "hermes-cli")
	endpoint, err := url.JoinPath(portalBaseURL, "/api/oauth/token")
	if err != nil {
		return nil, false
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", refreshToken)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebResponseBytes))
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	if strings.TrimSpace(webStringValue(payload["access_token"])) == "" {
		return nil, false
	}
	if strings.TrimSpace(webStringValue(payload["refresh_token"])) == "" {
		payload["refresh_token"] = refreshToken
	}
	if strings.TrimSpace(webStringValue(payload["token_type"])) == "" {
		payload["token_type"] = firstNonEmpty(webStringValue(provider["token_type"]), "Bearer")
	}
	return payload, true
}

func writeWebManagedAuthStore(pathValue string, original []byte, store webManagedAuthStore) error {
	var raw map[string]any
	if err := json.Unmarshal(original, &raw); err != nil {
		return err
	}
	providers, _ := raw["providers"].(map[string]any)
	if providers == nil {
		providers = map[string]any{}
		raw["providers"] = providers
	}
	for provider, state := range store.Providers {
		providers[provider] = state
	}
	body, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	mode := os.FileMode(0o600)
	if info, err := os.Stat(pathValue); err == nil {
		mode = info.Mode().Perm()
	}
	tmp := pathValue + ".tmp"
	if err := os.WriteFile(tmp, body, mode); err != nil {
		return err
	}
	return os.Rename(tmp, pathValue)
}

func webAuthTokenIsExpiring(expiresAt any, now time.Time) bool {
	parsed, ok := parseWebAuthTimestamp(webStringValue(expiresAt), now)
	if !ok {
		return true
	}
	return parsed.Sub(now.UTC()) <= webAuthRefreshSkew
}

func parseWebAuthTimestamp(value string, _ time.Time) (time.Time, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return time.Time{}, false
	}
	if strings.HasSuffix(normalized, "Z") {
		normalized = strings.TrimSuffix(normalized, "Z") + "+00:00"
	}
	parsed, err := time.Parse(time.RFC3339, normalized)
	if err != nil {
		if parsed, fallbackErr := time.Parse("2006-01-02T15:04:05", normalized); fallbackErr == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), time.UTC), true
		}
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func webStringValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	default:
		return ""
	}
}

func intValue(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case float64:
		return int(value)
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	default:
		return 0
	}
}

func webStringList(v any) []string {
	switch value := v.(type) {
	case []string:
		return value
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func parallelSearchMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PARALLEL_SEARCH_MODE"))) {
	case "fast":
		return "fast"
	case "one-shot":
		return "one-shot"
	case "agentic":
		return "agentic"
	default:
		return "agentic"
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func safeWebSnippet(raw []byte) string {
	const max = 512
	if len(raw) > max {
		raw = raw[:max]
	}
	return strings.TrimSpace(string(raw))
}

func redactWebError(message, secret string) string {
	out := message
	if secret != "" {
		out = strings.ReplaceAll(out, secret, "[redacted]")
	}
	if idx := strings.Index(strings.ToLower(out), "bearer "); idx >= 0 {
		end := idx + len("bearer ")
		for end < len(out) && out[end] != ' ' && out[end] != '\n' && out[end] != '\t' {
			end++
		}
		out = out[:idx+len("bearer ")] + "[redacted]" + out[end:]
	}
	return out
}
