package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/hub"
)

// HubSearchResult is a single entry returned by a registry provider while
// browsing the skills hub. Field names are stable across the gateway/RPC
// boundary so that downstream slices (Search, gateway dispatch) can rely on a
// wire-compatible shape.
type HubSearchResult = hub.HubSearchResult

// HubRegistryProvider yields a deterministic read-only snapshot of search
// results from a single registry source. Implementations must not mutate the
// active or inactive skill stores: the snapshot is a read-model used by the
// upcoming Search() function over multiple providers.
type HubRegistryProvider = hub.HubRegistryProvider

var (
	ErrRegistryUnavailable = hub.ErrRegistryUnavailable
	ErrRegistryRateLimited = hub.ErrRegistryRateLimited
	ErrRegistryMalformed   = hub.ErrRegistryMalformed
)

// InMemoryRegistryProvider is a deterministic test double that returns a
// preconfigured slice of HubSearchResult entries (sorted by Name ascending)
// or a preconfigured error. It is the only provider implementation in this
// slice; live registries land in later rows.
type InMemoryRegistryProvider = hub.InMemoryRegistryProvider

func NewInMemoryRegistryProvider(results []HubSearchResult, err error) *InMemoryRegistryProvider {
	return hub.NewInMemoryRegistryProvider(results, err)
}

// HubSearchEvidence is a typed enum reported by Search to describe degraded
// outcomes (empty query, unavailable registry, rate limited registry, no
// matches). Callers must inspect Evidence rather than assume an empty Results
// slice means failure.
type HubSearchEvidence = hub.HubSearchEvidence

const (
	HubSearchEvidenceEmptyQuery          HubSearchEvidence = hub.HubSearchEvidenceEmptyQuery
	HubSearchEvidenceRegistryUnavailable HubSearchEvidence = hub.HubSearchEvidenceRegistryUnavailable
	HubSearchEvidenceRateLimited         HubSearchEvidence = hub.HubSearchEvidenceRateLimited
	HubSearchEvidenceRegistryMalformed   HubSearchEvidence = hub.HubSearchEvidenceRegistryMalformed
	HubSearchEvidenceNoResults           HubSearchEvidence = hub.HubSearchEvidenceNoResults
)

type HubSearchOptions = hub.HubSearchOptions

type HubBrowseOptions = hub.HubBrowseOptions

type HubSearchResponse = hub.HubSearchResponse

type HubBrowseResponse = hub.HubBrowseResponse

func PreferHermesIndexProvider(ctx context.Context, index HubRegistryProvider, fallbacks []HubRegistryProvider) ([]HubRegistryProvider, HubSearchEvidence) {
	return hub.PreferHermesIndexProvider(ctx, index, fallbacks)
}

func Search(ctx context.Context, query string, providers []HubRegistryProvider, opts HubSearchOptions) (HubSearchResponse, error) {
	return hub.Search(ctx, query, providers, opts)
}

func Browse(ctx context.Context, providers []HubRegistryProvider, opts HubBrowseOptions) (HubBrowseResponse, error) {
	return hub.Browse(ctx, providers, opts)
}
