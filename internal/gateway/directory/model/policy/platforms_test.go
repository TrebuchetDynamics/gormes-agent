package policy

import "testing"

type bucketTestEntry struct {
	ID   string
	Name string
}

func TestEnsurePlatformBucketsInitializesNilMap(t *testing.T) {
	buckets := EnsurePlatformBuckets[bucketTestEntry](nil)
	if buckets == nil {
		t.Fatal("EnsurePlatformBuckets returned nil")
	}
	buckets["slack"] = []bucketTestEntry{{ID: "C01", Name: "ops"}}
	if got := buckets["slack"][0].Name; got != "ops" {
		t.Fatalf("bucket write = %q, want ops", got)
	}
}

func TestEnsurePlatformBucketsPreservesExistingMap(t *testing.T) {
	original := map[string][]bucketTestEntry{"telegram": {{ID: "chat", Name: "chat"}}}
	buckets := EnsurePlatformBuckets(original)
	buckets["telegram"][0].Name = "renamed"
	if original["telegram"][0].Name != "renamed" {
		t.Fatal("EnsurePlatformBuckets did not preserve existing map")
	}
}
