package model

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"

// trimText preserves the model package's internal normalization seam while the
// reusable policy contract lives in the focused policy subpackage.
func trimText(value string) string {
	return policy.TrimText(value)
}

// EmptyPlatformBuckets returns an initialized platform-indexed bucket map for
// directory read models and remembered-source ledgers.
func EmptyPlatformBuckets[T any]() map[string][]T {
	return policy.EmptyPlatformBuckets[T]()
}

// EnsurePlatformBuckets initializes a decoded platform-indexed bucket map when
// JSON omitted it or a zero value was used by a store.
func EnsurePlatformBuckets[T any](buckets map[string][]T) map[string][]T {
	return policy.EnsurePlatformBuckets(buckets)
}

// upsertByNormalizedID keeps the existing model-local helper name while
// reusing the shared replacement policy implementation.
func upsertByNormalizedID[T any](entries []T, entry T, normalize func(T) T, id func(T) string, valid func(T) bool) ([]T, bool) {
	return policy.UpsertByNormalizedID(entries, entry, normalize, id, valid)
}
