package policy

// EmptyPlatformBuckets returns an initialized platform-indexed bucket map for
// directory read models and remembered-source ledgers.
func EmptyPlatformBuckets[T any]() map[string][]T {
	return map[string][]T{}
}

// EnsurePlatformBuckets initializes a decoded platform-indexed bucket map when
// JSON omitted it or a zero value was used by a store.
func EnsurePlatformBuckets[T any](buckets map[string][]T) map[string][]T {
	if buckets == nil {
		return EmptyPlatformBuckets[T]()
	}
	return buckets
}
