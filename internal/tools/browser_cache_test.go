package tools

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBrowserResultCache_GetSet(t *testing.T) {
	cache := NewBrowserResultCache(5 * time.Minute)
	result := json.RawMessage(`{"text":"hello"}`)

	cache.Set(BrowserActionSnapshot, "https://example.com", result)

	got, ok := cache.Get(BrowserActionSnapshot, "https://example.com")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(result) {
		t.Fatalf("got %q, want %q", got, result)
	}

	_, ok = cache.Get(BrowserActionSnapshot, "https://other.com")
	if ok {
		t.Fatal("expected cache miss for different URL")
	}
}

func TestBrowserResultCache_TTLExpires(t *testing.T) {
	cache := NewBrowserResultCache(1 * time.Millisecond)
	result := json.RawMessage(`{"text":"hello"}`)
	cache.Set(BrowserActionSnapshot, "https://example.com", result)

	time.Sleep(5 * time.Millisecond)

	_, ok := cache.Get(BrowserActionSnapshot, "https://example.com")
	if ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestBrowserResultCache_Invalidate(t *testing.T) {
	cache := NewBrowserResultCache(5 * time.Minute)
	cache.Set(BrowserActionSnapshot, "https://example.com", json.RawMessage(`{}`))
	cache.Set(BrowserActionExtract, "https://example.com", json.RawMessage(`{}`))
	cache.Set(BrowserActionSnapshot, "https://other.com", json.RawMessage(`{}`))

	cache.Invalidate("https://example.com")

	if _, ok := cache.Get(BrowserActionSnapshot, "https://example.com"); ok {
		t.Fatal("expected snapshot invalidated")
	}
	if _, ok := cache.Get(BrowserActionExtract, "https://example.com"); ok {
		t.Fatal("expected extract invalidated")
	}
	if _, ok := cache.Get(BrowserActionSnapshot, "https://other.com"); !ok {
		t.Fatal("expected other URL still cached")
	}
}

func TestBrowserResultCache_InvalidateAll(t *testing.T) {
	cache := NewBrowserResultCache(5 * time.Minute)
	cache.Set(BrowserActionSnapshot, "https://a.com", json.RawMessage(`{}`))
	cache.Set(BrowserActionSnapshot, "https://b.com", json.RawMessage(`{}`))

	cache.InvalidateAll()

	if _, ok := cache.Get(BrowserActionSnapshot, "https://a.com"); ok {
		t.Fatal("expected all invalidated")
	}
}

func TestBrowserResultCache_DisabledWhenTTLZero(t *testing.T) {
	cache := NewBrowserResultCache(0)
	cache.Set(BrowserActionSnapshot, "https://example.com", json.RawMessage(`{}`))

	if _, ok := cache.Get(BrowserActionSnapshot, "https://example.com"); ok {
		t.Fatal("expected cache disabled when TTL is zero")
	}
}

func TestBrowserResultCache_Stats(t *testing.T) {
	cache := NewBrowserResultCache(5 * time.Minute)
	cache.Set(BrowserActionSnapshot, "https://a.com", json.RawMessage(`{}`))
	cache.Set(BrowserActionSnapshot, "https://b.com", json.RawMessage(`{}`))

	entries, ttl := cache.Stats()
	if entries != 2 {
		t.Fatalf("entries = %d, want 2", entries)
	}
	if ttl != 5*time.Minute {
		t.Fatalf("ttl = %v, want 5m", ttl)
	}
}

func TestIsCacheableBrowserAction(t *testing.T) {
	if !IsCacheableBrowserAction(BrowserActionSnapshot) {
		t.Fatal("snapshot should be cacheable")
	}
	if !IsCacheableBrowserAction(BrowserActionExtract) {
		t.Fatal("extract should be cacheable")
	}
	if IsCacheableBrowserAction(BrowserActionClick) {
		t.Fatal("click should not be cacheable")
	}
}

func TestIsInvalidatingBrowserAction(t *testing.T) {
	for _, action := range []string{BrowserActionClick, BrowserActionType, BrowserActionScroll, BrowserActionBack, BrowserActionNavigate} {
		if !IsInvalidatingBrowserAction(action) {
			t.Fatalf("%s should be invalidating", action)
		}
	}
	if IsInvalidatingBrowserAction(BrowserActionSnapshot) {
		t.Fatal("snapshot should not be invalidating")
	}
}
