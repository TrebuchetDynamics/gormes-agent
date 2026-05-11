package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// BrowserResultCache provides a TTL-gated cache for browser tool results keyed
// by URL and action kind. It avoids re-fetching the same page when multiple
// snapshot/extract actions target a recently-navigated URL.
type BrowserResultCache struct {
	ttl   time.Duration
	mu    sync.RWMutex
	items map[string]cachedBrowserResult
}

type cachedBrowserResult struct {
	Result    json.RawMessage
	CachedAt  time.Time
	ActionKind string
	URL        string
}

// NewBrowserResultCache creates a cache with the given TTL. A zero or negative
// TTL disables caching (all lookups miss and all stores are no-ops).
func NewBrowserResultCache(ttl time.Duration) *BrowserResultCache {
	return &BrowserResultCache{
		ttl:   ttl,
		items: map[string]cachedBrowserResult{},
	}
}

// cacheKey returns a deterministic key for an action kind + URL pair.
func cacheKey(actionKind, url string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(actionKind))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))
}

// Get returns a cached result if present and not expired. The second return
// value reports whether the cache entry was usable.
func (c *BrowserResultCache) Get(actionKind, url string) (json.RawMessage, bool) {
	if c == nil || c.ttl <= 0 {
		return nil, false
	}
	c.mu.RLock()
	item, ok := c.items[cacheKey(actionKind, url)]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(item.CachedAt) > c.ttl {
		return nil, false
	}
	return append(json.RawMessage(nil), item.Result...), true
}

// Set stores a result for the given action kind + URL.
func (c *BrowserResultCache) Set(actionKind, url string, result json.RawMessage) {
	if c == nil || c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[cacheKey(actionKind, url)] = cachedBrowserResult{
		Result:     append(json.RawMessage(nil), result...),
		CachedAt:   time.Now(),
		ActionKind: actionKind,
		URL:        url,
	}
}

// Invalidate removes all entries matching the given URL regardless of action
// kind. It is called after mutating browser actions (click, type, scroll, back,
// press) so subsequent snapshots reflect the new page state.
func (c *BrowserResultCache) Invalidate(url string) {
	if c == nil || c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.items {
		if v.URL == url {
			delete(c.items, k)
		}
	}
}

// InvalidateAll clears the entire cache.
func (c *BrowserResultCache) InvalidateAll() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]cachedBrowserResult{}
}

// Stats returns cache occupancy for operator diagnostics.
func (c *BrowserResultCache) Stats() (entries int, ttl time.Duration) {
	if c == nil {
		return 0, 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items), c.ttl
}

// IsCacheableAction reports whether a browser action kind supports result
// caching. Only idempotent read-style actions are cached.
func IsCacheableBrowserAction(kind string) bool {
	switch kind {
	case BrowserActionSnapshot, BrowserActionExtract:
		return true
	default:
		return false
	}
}

// IsInvalidatingAction reports whether a browser action kind should clear the
// cache for its URL because it mutates page state.
func IsInvalidatingBrowserAction(kind string) bool {
	switch kind {
	case BrowserActionClick, BrowserActionType, BrowserActionScroll,
		BrowserActionBack, BrowserActionNavigate:
		return true
	default:
		return false
	}
}

// BrowserCacheEvidence returns a concise operator-facing evidence string for
// cache hits/misses/invalidations.
func BrowserCacheEvidence(hit bool, actionKind, url string) string {
	if hit {
		return fmt.Sprintf("browser_cache_hit action=%s url=%s", actionKind, redactURLForEvidence(url))
	}
	return fmt.Sprintf("browser_cache_miss action=%s url=%s", actionKind, redactURLForEvidence(url))
}

func redactURLForEvidence(url string) string {
	if len(url) <= 40 {
		return url
	}
	return url[:20] + "..." + url[len(url)-20:]
}
