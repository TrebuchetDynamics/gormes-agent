package model

import "testing"

func TestEnsurePlatformBucketsInitializesNilMap(t *testing.T) {
	buckets := EnsurePlatformBuckets[Entry](nil)
	if buckets == nil {
		t.Fatal("EnsurePlatformBuckets returned nil")
	}
	buckets["slack"] = []Entry{{ID: "C01", Name: "ops"}}
	if got := buckets["slack"][0].Name; got != "ops" {
		t.Fatalf("bucket write = %q, want ops", got)
	}
}

func TestEnsurePlatformBucketsPreservesExistingMap(t *testing.T) {
	original := map[string][]RememberedSourceEntry{"telegram": {{ID: "chat", Name: "chat"}}}
	buckets := EnsurePlatformBuckets(original)
	buckets["telegram"][0].Name = "renamed"
	if original["telegram"][0].Name != "renamed" {
		t.Fatal("EnsurePlatformBuckets did not preserve existing map")
	}
}
