package resultcache

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBrowserResultCache_GetSet(t *testing.T) {
	cache := New(5 * time.Minute)
	result := json.RawMessage(`{"text":"hello"}`)

	cache.Set("snapshot", "https://example.com", result)

	got, ok := cache.Get("snapshot", "https://example.com")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(result) {
		t.Fatalf("got %q, want %q", got, result)
	}

	_, ok = cache.Get("snapshot", "https://other.com")
	if ok {
		t.Fatal("expected cache miss for different URL")
	}
}

func TestBrowserResultCache_TTLExpires(t *testing.T) {
	cache := New(1 * time.Millisecond)
	result := json.RawMessage(`{"text":"hello"}`)
	cache.Set("snapshot", "https://example.com", result)

	time.Sleep(5 * time.Millisecond)

	_, ok := cache.Get("snapshot", "https://example.com")
	if ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestBrowserResultCache_Invalidate(t *testing.T) {
	cache := New(5 * time.Minute)
	cache.Set("snapshot", "https://example.com", json.RawMessage(`{}`))
	cache.Set("extract", "https://example.com", json.RawMessage(`{}`))
	cache.Set("snapshot", "https://other.com", json.RawMessage(`{}`))

	cache.Invalidate("https://example.com")

	if _, ok := cache.Get("snapshot", "https://example.com"); ok {
		t.Fatal("expected snapshot invalidated")
	}
	if _, ok := cache.Get("extract", "https://example.com"); ok {
		t.Fatal("expected extract invalidated")
	}
	if _, ok := cache.Get("snapshot", "https://other.com"); !ok {
		t.Fatal("expected other URL still cached")
	}
}

func TestBrowserResultCache_InvalidateAll(t *testing.T) {
	cache := New(5 * time.Minute)
	cache.Set("snapshot", "https://a.com", json.RawMessage(`{}`))
	cache.Set("snapshot", "https://b.com", json.RawMessage(`{}`))

	cache.InvalidateAll()

	if _, ok := cache.Get("snapshot", "https://a.com"); ok {
		t.Fatal("expected all invalidated")
	}
}

func TestBrowserResultCache_DisabledWhenTTLZero(t *testing.T) {
	cache := New(0)
	cache.Set("snapshot", "https://example.com", json.RawMessage(`{}`))

	if _, ok := cache.Get("snapshot", "https://example.com"); ok {
		t.Fatal("expected cache disabled when TTL is zero")
	}
}

func TestBrowserResultCache_Stats(t *testing.T) {
	cache := New(5 * time.Minute)
	cache.Set("snapshot", "https://a.com", json.RawMessage(`{}`))
	cache.Set("snapshot", "https://b.com", json.RawMessage(`{}`))

	entries, ttl := cache.Stats()
	if entries != 2 {
		t.Fatalf("entries = %d, want 2", entries)
	}
	if ttl != 5*time.Minute {
		t.Fatalf("ttl = %v, want 5m", ttl)
	}
}

func TestIsCacheableAction(t *testing.T) {
	if !IsCacheableAction("snapshot") {
		t.Fatal("snapshot should be cacheable")
	}
	if !IsCacheableAction("extract") {
		t.Fatal("extract should be cacheable")
	}
	if IsCacheableAction("click") {
		t.Fatal("click should not be cacheable")
	}
}

func TestIsInvalidatingAction(t *testing.T) {
	for _, action := range []string{"click", "type", "scroll", "back", "navigate"} {
		if !IsInvalidatingAction(action) {
			t.Fatalf("%s should be invalidating", action)
		}
	}
	if IsInvalidatingAction("snapshot") {
		t.Fatal("snapshot should not be invalidating")
	}
}
