package model

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"

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
