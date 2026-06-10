package tools

import (
	"fmt"
	"os"
	"strings"
)

func ResolveWebBackend(env map[string]string) WebBackendResolution {
	return ResolveWebBackendWithConfig(env, WebBackendConfig{})
}

// ResolveWebBackendWithConfig resolves Hermes-compatible web.backend selection
// from explicit config first, then the same key-based fallback order as
// upstream Hermes: Firecrawl, Parallel, Tavily, Exa. CDP is a Gormes local
// extraction fallback for known URLs and is tried only after indexed providers.
func ResolveWebBackendWithConfig(env map[string]string, cfg WebBackendConfig) WebBackendResolution {
	read := func(key string) string {
		if env != nil {
			return strings.TrimSpace(env[key])
		}
		return strings.TrimSpace(os.Getenv(key))
	}

	if configured := normalizeWebBackend(cfg.Backend); configured != "" {
		return resolveConfiguredWebBackend(configured, cfg, read, true)
	}

	for _, backend := range []WebBackend{WebBackendFirecrawl, WebBackendParallel, WebBackendTavily, WebBackendExa, WebBackendBrave, WebBackendSearXNG, WebBackendPerplexity, WebBackendDuckDuckGo} {
		if resolved := resolveConfiguredWebBackend(backend, cfg, read, false); resolved.Available {
			return resolved
		}
	}

	return duckDuckGoBackendResolution(read)
}

func ResolveWebBackendStatus(env map[string]string, cfg WebBackendConfig) WebBackendStatus {
	resolved := ResolveWebBackendWithConfig(env, cfg)
	route := "unavailable"
	if resolved.Available {
		if resolved.Managed {
			route = "managed"
		} else if resolved.Backend == WebBackendCDP || resolved.Backend == WebBackendGoscraplingBrowser {
			route = "local"
		} else {
			route = "direct"
		}
	}
	description := "web provider unavailable"
	if resolved.Available {
		description = fmt.Sprintf("%s web backend via %s route", resolved.Backend, route)
	}
	return WebBackendStatus{
		Backend:     resolved.Backend,
		Available:   resolved.Available,
		Route:       route,
		Source:      firstNonEmpty(resolved.Source, route),
		BaseURL:     resolved.BaseURL,
		Evidence:    resolved.Evidence,
		UseGateway:  cfg.UseGateway,
		Managed:     resolved.Managed,
		ToolNames:   webBackendToolNames(resolved.Backend),
		RequiresEnv: webRequiresEnv(cfg.ManagedToolsEnabled),
		Description: description,
	}
}

func resolveConfiguredWebBackend(backend WebBackend, cfg WebBackendConfig, read func(string) string, explicit bool) WebBackendResolution {
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
				Source:    "env",
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
				Source:    "env",
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
				Source:    "env",
			}
		}
		return unavailableWebBackend(WebBackendExa, webDefaultExaBaseURL)
	case WebBackendBrave:
		if apiKey := read("BRAVE_API_KEY"); apiKey != "" {
			return WebBackendResolution{
				Backend:   WebBackendBrave,
				BaseURL:   normalizeWebBaseURL(firstNonEmpty(read("BRAVE_API_URL"), webDefaultBraveBaseURL)),
				APIKey:    apiKey,
				Available: true,
				Evidence:  WebEvidenceOK,
				Source:    "env",
			}
		}
		return unavailableWebBackend(WebBackendBrave, webDefaultBraveBaseURL)
	case WebBackendSearXNG:
		if baseURL := firstNonEmpty(read("SEARXNG_BASE_URL"), read("SEARXNG_URL")); baseURL != "" {
			return WebBackendResolution{
				Backend:   WebBackendSearXNG,
				BaseURL:   normalizeWebBaseURL(baseURL),
				Available: true,
				Evidence:  WebEvidenceOK,
				Source:    "env",
			}
		}
		return unavailableWebBackend(WebBackendSearXNG, "")
	case WebBackendPerplexity:
		if apiKey := read("PERPLEXITY_API_KEY"); apiKey != "" {
			return WebBackendResolution{
				Backend:   WebBackendPerplexity,
				BaseURL:   normalizeWebBaseURL(firstNonEmpty(read("PERPLEXITY_API_URL"), webDefaultPerplexityBaseURL)),
				APIKey:    apiKey,
				Available: true,
				Evidence:  WebEvidenceOK,
				Source:    "env",
			}
		}
		return unavailableWebBackend(WebBackendPerplexity, webDefaultPerplexityBaseURL)
	case WebBackendCDP:
		return resolveCDPBackend(read)
	case WebBackendGoscraplingBrowser:
		return resolveGoscraplingBrowserBackend(explicit)
	case WebBackendGoscraplingCrawler:
		return resolveGoscraplingCrawlerBackend(explicit)
	case WebBackendDuckDuckGo:
		return duckDuckGoBackendResolution(read)
	default:
		return unavailableWebBackend(WebBackendFirecrawl, webDefaultFirecrawlBaseURL)
	}
}

func duckDuckGoBackendResolution(read func(string) string) WebBackendResolution {
	return WebBackendResolution{
		Backend:   WebBackendDuckDuckGo,
		BaseURL:   normalizeWebBaseURL(firstNonEmpty(read("DUCKDUCKGO_BASE_URL"), read("DUCKDUCKGO_API_URL"), webDefaultDuckDuckGoBaseURL)),
		Available: true,
		Evidence:  WebEvidenceOK,
		Source:    "free",
		Note:      "free web_search via DuckDuckGo HTML/Lite parsing; web_extract fetches URLs locally with goscrapling and falls back to Instant Answer; no crawl support",
	}
}

func resolveCDPBackend(read func(string) string) WebBackendResolution {
	endpoint := strings.TrimSpace(firstNonEmpty(read("CHROME_REMOTE_DEBUGGING_URL"), read("BROWSER_CDP_URL")))
	if endpoint == "" {
		return unavailableWebBackend(WebBackendCDP, "")
	}
	return WebBackendResolution{
		Backend:   WebBackendCDP,
		BaseURL:   endpoint,
		Available: true,
		Evidence:  WebEvidenceOK,
		Source:    "env",
		Note:      "extract only — no search or crawl; pair with DuckDuckGo or SearXNG for search",
	}
}

func resolveGoscraplingBrowserBackend(explicit bool) WebBackendResolution {
	if !explicit {
		return unavailableWebBackend(WebBackendGoscraplingBrowser, "")
	}
	return WebBackendResolution{
		Backend:   WebBackendGoscraplingBrowser,
		Available: true,
		Evidence:  WebEvidenceOK,
		Source:    "config",
		Note:      "extract only — goscrapling browser renderer selected by web.backend",
	}
}

func resolveGoscraplingCrawlerBackend(explicit bool) WebBackendResolution {
	if !explicit {
		return unavailableWebBackend(WebBackendGoscraplingCrawler, "")
	}
	return WebBackendResolution{
		Backend:   WebBackendGoscraplingCrawler,
		Available: false,
		Evidence:  WebEvidenceProviderUnavailable,
		Source:    "config",
		Note:      "local goscrapling crawler is not yet available; use Firecrawl or Tavily until the crawler adapter gate is complete",
	}
}

func resolveFirecrawlBackend(cfg WebBackendConfig, read func(string) string) WebBackendResolution {
	apiURL := read("FIRECRAWL_API_URL")
	apiKey := read("FIRECRAWL_API_KEY")
	if (apiURL != "" || apiKey != "") && !cfg.UseGateway {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   normalizeWebBaseURL(firstNonEmpty(apiURL, webDefaultFirecrawlBaseURL)),
			APIKey:    apiKey,
			Available: true,
			Evidence:  WebEvidenceOK,
			Source:    "env",
		}
	}

	gatewayURL := read("FIRECRAWL_GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = managedFirecrawlGatewayURL(read("TOOL_GATEWAY_DOMAIN"), read("TOOL_GATEWAY_SCHEME"))
	}
	if !validWebGatewayURLScheme(gatewayURL) {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   gatewayURL,
			Available: false,
			Evidence:  WebEvidenceProviderUnavailable,
			Source:    "invalid_gateway_scheme",
		}
	}
	gatewayToken := read("TOOL_GATEWAY_USER_TOKEN")
	gatewaySource := "env"
	if gatewayToken == "" && cfg.ManagedToolsEnabled {
		if token, ok := readNousAccessTokenFromAuthStore(cfg.AuthStorePath, cfg.now(), cfg.authHTTPClient(), cfg.authRefreshTimeout()); ok {
			gatewayToken = token
			gatewaySource = "auth_store"
		}
	}
	if gatewayURL != "" && gatewayToken != "" && cfg.UseGateway {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   normalizeWebBaseURL(gatewayURL),
			APIKey:    gatewayToken,
			Available: true,
			Evidence:  WebEvidenceOK,
			Managed:   true,
			Source:    gatewaySource,
		}
	}

	if gatewayURL != "" && gatewayToken != "" {
		return WebBackendResolution{
			Backend:   WebBackendFirecrawl,
			BaseURL:   normalizeWebBaseURL(gatewayURL),
			APIKey:    gatewayToken,
			Available: true,
			Evidence:  WebEvidenceOK,
			Managed:   true,
			Source:    gatewaySource,
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
		Source:    "unavailable",
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
	case string(WebBackendGoscraplingBrowser), "goscrapling-browser", "browser_goscrapling":
		return WebBackendGoscraplingBrowser
	case string(WebBackendGoscraplingCrawler), "goscrapling-crawler", "crawler_goscrapling", "local_crawler", "local-crawler":
		return WebBackendGoscraplingCrawler
	case string(WebBackendCDP), "browser", "browser_cdp", "chrome":
		return WebBackendCDP
	case string(WebBackendBrave), "brave_search":
		return WebBackendBrave
	case string(WebBackendSearXNG), "searx":
		return WebBackendSearXNG
	case string(WebBackendPerplexity), "pplx", "perplexity_search":
		return WebBackendPerplexity
	case string(WebBackendDuckDuckGo), "ddg", "duckduckgo_search":
		return WebBackendDuckDuckGo
	default:
		return ""
	}
}
