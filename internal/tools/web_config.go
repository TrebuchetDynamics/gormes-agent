package tools

import (
	"net/http"
	"time"
)

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

func (cfg WebBackendConfig) now() time.Time {
	if cfg.Now != nil {
		return cfg.Now()
	}
	return time.Now()
}

func (cfg WebBackendConfig) authHTTPClient() WebHTTPClient {
	if cfg.AuthHTTPClient != nil {
		return cfg.AuthHTTPClient
	}
	return http.DefaultClient
}

func (cfg WebBackendConfig) authRefreshTimeout() time.Duration {
	if cfg.AuthRefreshTimeout > 0 {
		return cfg.AuthRefreshTimeout
	}
	return webAuthRefreshTimeout
}
