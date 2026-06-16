package browser

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser/resultcache"
)

// BrowserResultCache provides a TTL-gated cache for browser tool results keyed
// by URL and action kind. It avoids re-fetching the same page when multiple
// snapshot/extract actions target a recently-navigated URL.
type BrowserResultCache = resultcache.Cache

// NewBrowserResultCache creates a cache with the given TTL. A zero or negative
// TTL disables caching (all lookups miss and all stores are no-ops).
func NewBrowserResultCache(ttl time.Duration) *BrowserResultCache {
	return resultcache.New(ttl)
}

// IsCacheableBrowserAction reports whether a browser action kind supports result
// caching. Only idempotent read-style actions are cached.
func IsCacheableBrowserAction(kind string) bool {
	return resultcache.IsCacheableAction(kind)
}

// IsInvalidatingBrowserAction reports whether a browser action kind should clear
// the cache for its URL because it mutates page state.
func IsInvalidatingBrowserAction(kind string) bool {
	return resultcache.IsInvalidatingAction(kind)
}

// BrowserCacheEvidence returns a concise operator-facing evidence string for
// cache hits/misses/invalidations.
func BrowserCacheEvidence(hit bool, actionKind, url string) string {
	return resultcache.Evidence(hit, actionKind, url)
}
