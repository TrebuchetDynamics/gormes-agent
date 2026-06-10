package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/TrebuchetDynamics/goscrapling"
	goscraplingbrowser "github.com/TrebuchetDynamics/goscrapling/engines/browser"
)

const (
	WebToolExtract = "web_extract"
	WebToolSearch  = "web_search"
	WebToolCrawl   = "web_crawl"

	webDefaultFirecrawlBaseURL  = "https://api.firecrawl.dev"
	webDefaultExaBaseURL        = "https://api.exa.ai"
	webDefaultParallelBaseURL   = "https://api.parallel.ai"
	webDefaultTavilyBaseURL     = "https://api.tavily.com"
	webDefaultBraveBaseURL      = "https://api.search.brave.com"
	webDefaultPerplexityBaseURL = "https://api.perplexity.ai"
	webDefaultDuckDuckGoBaseURL = "https://html.duckduckgo.com"
	webDefaultUserAgent         = "Mozilla/5.0 (compatible; GormesAgent/1.0)"
	defaultWebTimeout           = 60 * time.Second
	defaultWebSearchLimit       = 5
	defaultWebMaxSearch         = 100
	defaultWebMaxExtract        = 5
	defaultWebCrawlLimit        = 20
	defaultWebProcessMinLength  = 5000
	defaultWebProcessMaxInput   = 2_000_000
	defaultWebProcessMaxOutput  = 5000
	maxWebResponseBytes         = 2 * 1024 * 1024
	webAuthRefreshSkew          = 120 * time.Second
	webAuthRefreshTimeout       = 15 * time.Second
)

type WebBackend string

const (
	WebBackendFirecrawl          WebBackend = "firecrawl"
	WebBackendParallel           WebBackend = "parallel"
	WebBackendTavily             WebBackend = "tavily"
	WebBackendExa                WebBackend = "exa"
	WebBackendCDP                WebBackend = "cdp"
	WebBackendBrave              WebBackend = "brave"
	WebBackendSearXNG            WebBackend = "searxng"
	WebBackendPerplexity         WebBackend = "perplexity"
	WebBackendDuckDuckGo         WebBackend = "duckduckgo"
	WebBackendGoscraplingBrowser WebBackend = "goscrapling_browser"
	WebBackendGoscraplingCrawler WebBackend = "goscrapling_crawler"
)

type WebEvidence string

const (
	WebEvidenceOK                  WebEvidence = "web_ok"
	WebEvidenceProviderUnavailable WebEvidence = "web_provider_unavailable"
	WebEvidenceInvalidArguments    WebEvidence = "web_invalid_arguments"
	WebEvidenceRequestFailed       WebEvidence = "web_provider_request_failed"
	WebEvidenceBackendUnsupported  WebEvidence = "web_backend_unsupported"
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

type GoscraplingBrowserFetcher interface {
	Fetch(context.Context, string, goscraplingbrowser.BrowserOptions) (*goscrapling.Response, error)
}

type GoscraplingBrowserConfig struct {
	Fetcher GoscraplingBrowserFetcher
	Timeout time.Duration
	Wait    time.Duration
}

type GoscraplingCrawler interface {
	Crawl(context.Context, GoscraplingCrawlRequest) (GoscraplingCrawlResult, error)
}

type GoscraplingCrawlerConfig struct {
	Crawler  GoscraplingCrawler
	MaxPages int
}

type GoscraplingCrawlRequest struct {
	URL          string
	Instructions string
	Depth        string
	MaxPages     int
}

type GoscraplingCrawlResult struct {
	Pages []GoscraplingCrawlPage
	Stats GoscraplingCrawlStats
}

type GoscraplingCrawlPage struct {
	URL         string
	FinalURL    string
	Title       string
	Content     string
	StatusCode  int
	ContentType string
	Error       string
	Evidence    WebEvidence
	Duplicate   bool
	Offsite     bool
}

type GoscraplingCrawlStats struct {
	Visited    int `json:"visited,omitempty"`
	Duplicates int `json:"duplicates,omitempty"`
	Offsite    int `json:"offsite,omitempty"`
	MaxPages   int `json:"max_pages,omitempty"`
}

// WebBackendResolution is the credential/provider state used by web tools.
// Secret values stay in memory only long enough to attach request headers.
type WebBackendResolution struct {
	Backend   WebBackend
	BaseURL   string
	APIKey    string
	Available bool
	Evidence  WebEvidence
	Managed   bool
	Source    string
	Note      string
}

// WebBackendStatus is the redacted operator/toolset read model for native web
// tools. It intentionally excludes bearer tokens and raw credential values.
type WebBackendStatus struct {
	Backend     WebBackend  `json:"backend"`
	Available   bool        `json:"available"`
	Route       string      `json:"route"`
	Source      string      `json:"source"`
	BaseURL     string      `json:"base_url"`
	Evidence    WebEvidence `json:"evidence"`
	UseGateway  bool        `json:"use_gateway"`
	Managed     bool        `json:"managed"`
	ToolNames   []string    `json:"tool_names"`
	RequiresEnv []string    `json:"requires_env"`
	Description string      `json:"description"`
}

// WebBackendConfig mirrors Hermes' web config.yaml surface for backend
// selection. Empty Backend means auto-detect from available credentials.
type WebBackendConfig struct {
	Backend             string
	UseGateway          bool
	ManagedToolsEnabled bool
	AuthStorePath       string
	AuthHTTPClient      WebHTTPClient
	AuthRefreshTimeout  time.Duration
	Now                 func() time.Time
}

// WebToolsConfig wires web_search, web_extract, and web_crawl. Leave Resolution
// zero in production to resolve from environment; tests provide explicit values.
type WebToolsConfig struct {
	Client             WebHTTPClient
	Browser            BrowserHarnessToolsConfig
	Backend            WebBackendConfig
	GoscraplingBrowser GoscraplingBrowserConfig
	GoscraplingCrawler GoscraplingCrawlerConfig
	Policy             WebWebsitePolicy
	Processing         WebContentProcessingConfig
	ContentProcessor   WebContentProcessor
	Resolution         WebBackendResolution
	Timeout            time.Duration
	MaxSearch          int
	MaxExtract         int
	DefaultLimit       int
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
	Error          string      `json:"error,omitempty"`
	Evidence       WebEvidence `json:"evidence,omitempty"`
	Backend        WebBackend  `json:"backend,omitempty"`
	Source         string      `json:"source,omitempty"`
	Degraded       bool        `json:"degraded,omitempty"`
	DegradedReason string      `json:"degraded_reason,omitempty"`
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
	Extraction      *webExtraction    `json:"extraction,omitempty"`
	BlockedByPolicy map[string]string `json:"blocked_by_policy,omitempty"`
}

type webExtraction struct {
	Engine       string                 `json:"engine,omitempty"`
	Mode         string                 `json:"mode,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ContentType  string                 `json:"content_type,omitempty"`
	CSSSelector  string                 `json:"css_selector,omitempty"`
	FinalURL     string                 `json:"final_url,omitempty"`
	WaitEvidence string                 `json:"wait_evidence,omitempty"`
	Crawl        *GoscraplingCrawlStats `json:"crawl,omitempty"`
}

type webErrorResponse struct {
	Success        bool        `json:"success"`
	Error          string      `json:"error"`
	Evidence       WebEvidence `json:"evidence,omitempty"`
	Backend        WebBackend  `json:"backend,omitempty"`
	Degraded       bool        `json:"degraded,omitempty"`
	DegradedReason string      `json:"degraded_reason,omitempty"`
}

// ResolveWebBackend resolves the currently usable web backend from a provided
// environment map. The Go-native slice supports Firecrawl direct or
// self-hosted endpoints first; managed gateway credentials intentionally map
// into the same resolution shape so the future gateway adapter can reuse the
