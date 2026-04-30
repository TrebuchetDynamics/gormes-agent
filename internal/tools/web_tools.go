package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	WebToolExtract = "web_extract"
	WebToolSearch  = "web_search"

	webDefaultFirecrawlBaseURL = "https://api.firecrawl.dev"
	webDefaultExaBaseURL       = "https://api.exa.ai"
	webDefaultParallelBaseURL  = "https://api.parallel.ai"
	webDefaultTavilyBaseURL    = "https://api.tavily.com"
	defaultWebTimeout          = 60 * time.Second
	defaultWebSearchLimit      = 5
	defaultWebMaxSearch        = 100
	defaultWebMaxExtract       = 5
	defaultWebProcessMinLength = 5000
	defaultWebProcessMaxInput  = 2_000_000
	defaultWebProcessMaxOutput = 5000
	maxWebResponseBytes        = 2 * 1024 * 1024
)

type WebBackend string

const (
	WebBackendFirecrawl WebBackend = "firecrawl"
	WebBackendParallel  WebBackend = "parallel"
	WebBackendTavily    WebBackend = "tavily"
	WebBackendExa       WebBackend = "exa"
)

type WebEvidence string

const (
	WebEvidenceOK                  WebEvidence = "web_ok"
	WebEvidenceProviderUnavailable WebEvidence = "web_provider_unavailable"
	WebEvidenceInvalidArguments    WebEvidence = "web_invalid_arguments"
	WebEvidenceRequestFailed       WebEvidence = "web_provider_request_failed"
	WebEvidencePrivateURLBlocked   WebEvidence = "private_url_blocked"
	WebEvidenceWebsitePolicy       WebEvidence = "website_policy_blocked"
	WebEvidenceSecretURLBlocked    WebEvidence = "secret_url_blocked"
)

// WebHTTPClient is the only network boundary for the native web tools. Tests
// inject fakes; production uses http.DefaultClient until managed gateway
// wiring supplies its own adapter.
type WebHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// WebBackendResolution is the credential/provider state used by web tools.
// Secret values stay in memory only long enough to attach request headers.
type WebBackendResolution struct {
	Backend   WebBackend
	BaseURL   string
	APIKey    string
	Available bool
	Evidence  WebEvidence
}

// WebBackendConfig mirrors Hermes' web config.yaml surface for backend
// selection. Empty Backend means auto-detect from available credentials.
type WebBackendConfig struct {
	Backend    string
	UseGateway bool
}

// WebToolsConfig wires web_search and web_extract. Leave Resolution zero in
// production to resolve from environment; tests provide explicit values.
type WebToolsConfig struct {
	Client           WebHTTPClient
	Backend          WebBackendConfig
	Policy           WebWebsitePolicy
	Processing       WebContentProcessingConfig
	ContentProcessor WebContentProcessor
	Resolution       WebBackendResolution
	Timeout          time.Duration
	MaxSearch        int
	MaxExtract       int
	DefaultLimit     int
}

// WebWebsitePolicy mirrors Hermes' security.website_blocklist policy in the
// part web tools need at execution time.
type WebWebsitePolicy struct {
	Enabled           bool
	Domains           []string
	SharedFiles       []string
	SharedFileBaseDir string
}

type WebContentProcessingConfig struct {
	Enabled        bool
	MinLength      int
	MaxInputChars  int
	MaxOutputChars int
}

type WebContentProcessRequest struct {
	URL     string
	Title   string
	Content string
}

type WebContentProcessor interface {
	ProcessWebContent(context.Context, WebContentProcessRequest) (string, error)
}

type webTool struct {
	name   string
	desc   string
	schema json.RawMessage
	cfg    WebToolsConfig
}

type webSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Position    int    `json:"position"`
}

type webSearchResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Web []webSearchResult `json:"web"`
	} `json:"data"`
	Error    string      `json:"error,omitempty"`
	Evidence WebEvidence `json:"evidence,omitempty"`
}

type webExtractResponse struct {
	Results []webExtractResult `json:"results"`
}

type webExtractResult struct {
	URL             string            `json:"url"`
	Title           string            `json:"title"`
	Content         string            `json:"content"`
	Error           string            `json:"error,omitempty"`
	Evidence        WebEvidence       `json:"evidence,omitempty"`
	BlockedByPolicy map[string]string `json:"blocked_by_policy,omitempty"`
}

type webErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// ResolveWebBackend resolves the currently usable web backend from a provided
// environment map. The Go-native slice supports Firecrawl direct or
// self-hosted endpoints first; managed gateway credentials intentionally map
// into the same resolution shape so the future gateway adapter can reuse the
// public tool contract.
func ResolveWebBackend(env map[string]string) WebBackendResolution {
	return ResolveWebBackendWithConfig(env, WebBackendConfig{})
}

// ResolveWebBackendWithConfig resolves Hermes-compatible web.backend selection
// from explicit config first, then the same key-based fallback order as
// upstream Hermes: Firecrawl, Parallel, Tavily, Exa.
func ResolveWebBackendWithConfig(env map[string]string, cfg WebBackendConfig) WebBackendResolution {
	read := func(key string) string {
		if env != nil {
			return strings.TrimSpace(env[key])
		}
		return strings.TrimSpace(os.Getenv(key))
	}

	if configured := normalizeWebBackend(cfg.Backend); configured != "" {
		return resolveConfiguredWebBackend(configured, cfg, read)
	}

	for _, backend := range []WebBackend{WebBackendFirecrawl, WebBackendParallel, WebBackendTavily, WebBackendExa} {
		if resolved := resolveConfiguredWebBackend(backend, cfg, read); resolved.Available {
			return resolved
		}
	}

	return WebBackendResolution{
		Backend:   WebBackendFirecrawl,
		BaseURL:   webDefaultFirecrawlBaseURL,
		Available: false,
		Evidence:  WebEvidenceProviderUnavailable,
	}
}

func resolveConfiguredWebBackend(backend WebBackend, cfg WebBackendConfig, read func(string) string) WebBackendResolution {
	switch backend {
	case WebBackendFirecrawl:
		return resolveFirecrawlBackend(cfg, read)
	case WebBackendParallel:
		if apiKey := read("PARALLEL_API_KEY"); apiKey != "" {
			return WebBackendResolution{
				Backend:   WebBackendParallel,
				BaseURL:   normalizeWebBaseURL(firstNonEmpty(read("PARALLEL_API_URL"), webDefaultParallelBaseURL)),
				APIKey:    apiKey,
				Available: true,
				Evidence:  WebEvidenceOK,
			}
		}
		return unavailableWebBackend(WebBackendParallel, webDefaultParallelBaseURL)
	case WebBackendTavily:
		if apiKey := read("TAVILY_API_KEY"); apiKey != "" {
			return WebBackendResolution{
				Backend:   WebBackendTavily,
				BaseURL:   normalizeWebBaseURL(firstNonEmpty(read("TAVILY_BASE_URL"), webDefaultTavilyBaseURL)),
				APIKey:    apiKey,
				Available: true,
				Evidence:  WebEvidenceOK,
			}
		}
		return unavailableWebBackend(WebBackendTavily, webDefaultTavilyBaseURL)
	case WebBackendExa:
		if apiKey := read("EXA_API_KEY"); apiKey != "" {
			return WebBackendResolution{
				Backend:   WebBackendExa,
				BaseURL:   normalizeWebBaseURL(firstNonEmpty(read("EXA_API_URL"), webDefaultExaBaseURL)),
				APIKey:    apiKey,
				Available: true,
				Evidence:  WebEvidenceOK,
			}
		}
		return unavailableWebBackend(WebBackendExa, webDefaultExaBaseURL)
	default:
		return unavailableWebBackend(WebBackendFirecrawl, webDefaultFirecrawlBaseURL)
	}
}

func resolveFirecrawlBackend(cfg WebBackendConfig, read func(string) string) WebBackendResolution {
	gatewayURL := read("FIRECRAWL_GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = managedFirecrawlGatewayURL(read("TOOL_GATEWAY_DOMAIN"), read("TOOL_GATEWAY_SCHEME"))
	}
	gatewayToken := read("TOOL_GATEWAY_USER_TOKEN")
	if gatewayURL != "" && gatewayToken != "" && cfg.UseGateway {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   normalizeWebBaseURL(gatewayURL),
			APIKey:    gatewayToken,
			Available: true,
			Evidence:  WebEvidenceOK,
		}
	}

	apiURL := read("FIRECRAWL_API_URL")
	apiKey := read("FIRECRAWL_API_KEY")
	if (apiURL != "" || apiKey != "") && !cfg.UseGateway {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   normalizeWebBaseURL(firstNonEmpty(apiURL, webDefaultFirecrawlBaseURL)),
			APIKey:    apiKey,
			Available: true,
			Evidence:  WebEvidenceOK,
		}
	}

	if gatewayURL != "" && gatewayToken != "" {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   normalizeWebBaseURL(gatewayURL),
			APIKey:    gatewayToken,
			Available: true,
			Evidence:  WebEvidenceOK,
		}
	}

	return unavailableWebBackend(WebBackendFirecrawl, webDefaultFirecrawlBaseURL)
}

func unavailableWebBackend(backend WebBackend, baseURL string) WebBackendResolution {
	return WebBackendResolution{
		Backend:   backend,
		BaseURL:   baseURL,
		Available: false,
		Evidence:  WebEvidenceProviderUnavailable,
	}
}

func normalizeWebBackend(raw string) WebBackend {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(WebBackendFirecrawl):
		return WebBackendFirecrawl
	case string(WebBackendParallel):
		return WebBackendParallel
	case string(WebBackendTavily):
		return WebBackendTavily
	case string(WebBackendExa):
		return WebBackendExa
	default:
		return ""
	}
}

func NewWebTools(cfg WebToolsConfig) []Tool {
	return []Tool{
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
		schema: json.RawMessage(`{"type":"object","properties":{"urls":{"type":"array","items":{"type":"string"},"maxItems":5,"description":"List of URLs to extract content from (max 5 URLs per call)"}},"required":["urls"]}`),
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
		return webSearchFailure("Web tools are not configured. Set FIRECRAWL_API_KEY or FIRECRAWL_API_URL.", WebEvidenceProviderUnavailable)
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
	return json.Marshal(normalizeWebSearch(raw))
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
	default:
		return t.postFirecrawlJSON(ctx, "search", map[string]any{
			"query": query,
			"limit": limit,
		})
	}
}

func (t *webTool) executeExtract(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		URLs   []string `json:"urls"`
		Format string   `json:"format"`
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
	if !t.cfg.Resolution.Available {
		return json.Marshal(webExtractResponse{Results: []webExtractResult{{
			Error:    "Web tools are not configured. Set FIRECRAWL_API_KEY or FIRECRAWL_API_URL.",
			Evidence: WebEvidenceProviderUnavailable,
		}}})
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
			results = append(results, webExtractResult{
				URL:      trimmed,
				Error:    "Error extracting web page: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
				Evidence: WebEvidenceRequestFailed,
			})
			continue
		}
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(normalizeWebExtract(trimmed, in.Format, raw))))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
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
			results = append(results, webExtractResult{
				URL:      url,
				Error:    "Error extracting web page: " + redactWebError(err.Error(), t.cfg.Resolution.APIKey),
				Evidence: WebEvidenceRequestFailed,
			})
		}
		return marshalWebExtractResponse(webExtractResponse{Results: results})
	}
	for _, result := range normalizeWebExtractDocuments(safeURLs, format, raw) {
		results = append(results, t.processWebExtractResult(ctx, t.applyWebExtractPostPolicy(result)))
	}
	return marshalWebExtractResponse(webExtractResponse{Results: results})
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
	default:
		return nil, fmt.Errorf("web: unsupported extract backend %q", t.cfg.Resolution.Backend)
	}
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

func normalizeWebToolsConfig(cfg WebToolsConfig) WebToolsConfig {
	if cfg.Resolution.Backend == "" && cfg.Resolution.BaseURL == "" && !cfg.Resolution.Available && cfg.Resolution.Evidence == "" {
		cfg.Resolution = ResolveWebBackendWithConfig(nil, cfg.Backend)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultWebTimeout
	}
	return cfg
}

func (cfg WebToolsConfig) client() WebHTTPClient {
	if cfg.Client != nil {
		return cfg.Client
	}
	return http.DefaultClient
}

func (cfg WebToolsConfig) timeout() time.Duration {
	if cfg.Timeout > 0 {
		return cfg.Timeout
	}
	return defaultWebTimeout
}

func (cfg WebToolsConfig) maxSearch() int {
	if cfg.MaxSearch > 0 {
		return cfg.MaxSearch
	}
	return defaultWebMaxSearch
}

func (cfg WebToolsConfig) maxExtract() int {
	if cfg.MaxExtract > 0 {
		return cfg.MaxExtract
	}
	return defaultWebMaxExtract
}

func (cfg WebToolsConfig) defaultLimit() int {
	if cfg.DefaultLimit > 0 {
		return cfg.DefaultLimit
	}
	return defaultWebSearchLimit
}

func (cfg WebToolsConfig) processMinLength() int {
	if cfg.Processing.MinLength > 0 {
		return cfg.Processing.MinLength
	}
	return defaultWebProcessMinLength
}

func (cfg WebToolsConfig) processMaxInput() int {
	if cfg.Processing.MaxInputChars > 0 {
		return cfg.Processing.MaxInputChars
	}
	return defaultWebProcessMaxInput
}

func (cfg WebToolsConfig) processMaxOutput() int {
	if cfg.Processing.MaxOutputChars > 0 {
		return cfg.Processing.MaxOutputChars
	}
	return defaultWebProcessMaxOutput
}

func normalizeWebSearch(raw map[string]any) webSearchResponse {
	out := webSearchResponse{Success: true}
	candidates := webSearchCandidates(raw)
	out.Data.Web = make([]webSearchResult, 0, len(candidates))
	for i, item := range candidates {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		position := intValue(row["position"])
		if position <= 0 {
			position = i + 1
		}
		out.Data.Web = append(out.Data.Web, webSearchResult{
			Title:       webStringValue(row["title"]),
			URL:         webStringValue(row["url"]),
			Description: firstNonEmpty(webStringValue(row["description"]), webStringValue(row["content"]), webStringValue(row["snippet"]), strings.Join(webStringList(row["highlights"]), " "), strings.Join(webStringList(row["excerpts"]), " ")),
			Position:    position,
		})
	}
	if success, ok := raw["success"].(bool); ok {
		out.Success = success
	}
	if !out.Success {
		out.Error = webStringValue(raw["error"])
		out.Evidence = WebEvidenceRequestFailed
	}
	return out
}

func normalizeWebExtract(requestedURL, format string, raw map[string]any) webExtractResult {
	payload := raw
	if data, ok := raw["data"].(map[string]any); ok {
		payload = data
	}
	metadata, _ := payload["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	finalURL := firstNonEmpty(webStringValue(metadata["sourceURL"]), webStringValue(payload["url"]), requestedURL)
	title := firstNonEmpty(webStringValue(metadata["title"]), webStringValue(payload["title"]))
	content := webExtractContent(payload, format)
	result := webExtractResult{
		URL:     finalURL,
		Title:   title,
		Content: content,
	}
	if errMsg := webStringValue(payload["error"]); errMsg != "" {
		result.Error = errMsg
		result.Evidence = WebEvidenceRequestFailed
	}
	return result
}

func normalizeWebExtractDocuments(requestedURLs []string, format string, raw map[string]any) []webExtractResult {
	var rows []map[string]any
	if results := mapList(raw["results"]); len(results) > 0 {
		rows = append(rows, results...)
	}
	if content, ok := raw["content"].(map[string]any); ok {
		rows = append(rows, content)
	}
	out := make([]webExtractResult, 0, len(rows))
	for i, row := range rows {
		fallbackURL := ""
		if i < len(requestedURLs) {
			fallbackURL = requestedURLs[i]
		}
		out = append(out, normalizeWebExtractDocument(row, fallbackURL, format))
	}
	for _, fail := range mapList(raw["failed_results"]) {
		out = append(out, webExtractResult{
			URL:   firstNonEmpty(webStringValue(fail["url"]), firstRequestedURL(requestedURLs)),
			Error: firstNonEmpty(webStringValue(fail["error"]), "extraction failed"),
		})
	}
	for _, failURL := range webStringList(raw["failed_urls"]) {
		out = append(out, webExtractResult{
			URL:   failURL,
			Error: "extraction failed",
		})
	}
	return out
}

func normalizeWebExtractDocument(row map[string]any, fallbackURL string, format string) webExtractResult {
	metadata, _ := row["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	content := webExtractContent(row, format)
	if content == "" {
		content = firstNonEmpty(webStringValue(row["text"]), webStringValue(row["raw_content"]), webStringValue(row["full_content"]), strings.Join(webStringList(row["excerpts"]), "\n\n"))
	}
	result := webExtractResult{
		URL:     firstNonEmpty(webStringValue(row["url"]), webStringValue(metadata["sourceURL"]), fallbackURL),
		Title:   firstNonEmpty(webStringValue(row["title"]), webStringValue(metadata["title"])),
		Content: content,
	}
	if errMsg := webStringValue(row["error"]); errMsg != "" {
		result.Error = errMsg
		result.Evidence = WebEvidenceRequestFailed
	}
	return result
}

func webSearchCandidates(raw map[string]any) []any {
	if data, ok := raw["data"].([]any); ok {
		return data
	}
	if data, ok := raw["data"].(map[string]any); ok {
		if web, ok := data["web"].([]any); ok {
			return web
		}
		if results, ok := data["results"].([]any); ok {
			return results
		}
	}
	if web, ok := raw["web"].([]any); ok {
		return web
	}
	if results, ok := raw["results"].([]any); ok {
		return results
	}
	return nil
}

func mapList(v any) []map[string]any {
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

var webBase64ImagePattern = regexp.MustCompile(`\(?data:image/[^;()\s]+;base64,[A-Za-z0-9+/=]+\)?`)

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
	if domain == "" {
		return ""
	}
	scheme = strings.TrimSpace(scheme)
	if scheme == "" {
		scheme = "https"
	}
	if strings.Contains(domain, "://") {
		return domain
	}
	return scheme + "://firecrawl-gateway." + strings.TrimPrefix(domain, ".")
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
