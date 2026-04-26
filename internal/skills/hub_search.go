package skills

import (
	"context"
	"errors"
	"sort"
)

// HubSearchResult is a single entry returned by a registry provider while
// browsing the skills hub. Field names are stable across the gateway/RPC
// boundary so that downstream slices (Search, gateway dispatch) can rely on a
// wire-compatible shape.
type HubSearchResult struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Source      string  `json:"source"`
	InstallID   string  `json:"install_id"`
	Score       float64 `json:"score"`
}

// HubRegistryProvider yields a deterministic read-only snapshot of search
// results from a single registry source. Implementations must not mutate the
// active or inactive skill stores: the snapshot is a read-model used by the
// upcoming Search() function over multiple providers.
type HubRegistryProvider interface {
	Snapshot(ctx context.Context) ([]HubSearchResult, error)
}

// Sentinel errors so downstream slices can table-test degraded evidence
// without depending on string matching or live network behaviour. The text
// matches the wire codes used by the future HubSearchResponse.Evidence field.
var (
	ErrRegistryUnavailable = errors.New("registry_unavailable")
	ErrRegistryRateLimited = errors.New("registry_rate_limited")
)

// InMemoryRegistryProvider is a deterministic test double that returns a
// preconfigured slice of HubSearchResult entries (sorted by Name ascending)
// or a preconfigured error. It is the only provider implementation in this
// slice; live registries land in later rows.
type InMemoryRegistryProvider struct {
	results []HubSearchResult
	err     error
}

// NewInMemoryRegistryProvider returns a provider that yields a defensive copy
// of the given results sorted by Name ascending. If err is non-nil, Snapshot
// returns it unchanged and the results slice is ignored on the read path.
func NewInMemoryRegistryProvider(results []HubSearchResult, err error) *InMemoryRegistryProvider {
	sorted := make([]HubSearchResult, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return &InMemoryRegistryProvider{results: sorted, err: err}
}

// Snapshot returns the configured error when set, otherwise a fresh copy of
// the deterministic results slice. The copy prevents callers from mutating
// the provider's view between calls.
func (p *InMemoryRegistryProvider) Snapshot(_ context.Context) ([]HubSearchResult, error) {
	if p == nil {
		return nil, nil
	}
	if p.err != nil {
		return nil, p.err
	}
	out := make([]HubSearchResult, len(p.results))
	copy(out, p.results)
	return out, nil
}
