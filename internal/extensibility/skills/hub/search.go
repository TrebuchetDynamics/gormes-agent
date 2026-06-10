package hub

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// HubSearchResult is a single entry returned by a registry provider while
// browsing the skills hub. Field names are stable across the gateway/RPC
// boundary so that downstream slices (Search, gateway dispatch) can rely on a
// wire-compatible shape.
type HubSearchResult struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	InstallID   string   `json:"install_id"`
	Score       float64  `json:"score"`
	TrustLevel  string   `json:"trust_level,omitempty"`
	Tags        []string `json:"tags,omitempty"`
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
	ErrRegistryMalformed   = errors.New("registry_malformed")
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
	sorted := cloneHubSearchResults(results)
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
	return cloneHubSearchResults(p.results), nil
}

func cloneHubSearchResults(results []HubSearchResult) []HubSearchResult {
	out := make([]HubSearchResult, len(results))
	copy(out, results)
	for i := range out {
		out[i].Tags = append([]string(nil), out[i].Tags...)
	}
	return out
}

// HubSearchEvidence is a typed enum reported by Search to describe degraded
// outcomes (empty query, unavailable registry, rate limited registry, no
// matches). Callers must inspect Evidence rather than assume an empty Results
// slice means failure.
type HubSearchEvidence string

const (
	HubSearchEvidenceEmptyQuery          HubSearchEvidence = "empty_query"
	HubSearchEvidenceRegistryUnavailable HubSearchEvidence = "registry_unavailable"
	HubSearchEvidenceRateLimited         HubSearchEvidence = "registry_rate_limited"
	HubSearchEvidenceRegistryMalformed   HubSearchEvidence = "registry_malformed"
	HubSearchEvidenceNoResults           HubSearchEvidence = "no_results"
)

// HubSearchOptions are the read-side options accepted by Search. Reserved for
// downstream slices that need source filters or limits; the current row has
// no required behaviour for any field.
type HubSearchOptions struct {
	// Limit caps the merged result list after dedupe and sort. Zero means no cap.
	Limit int
}

// HubBrowseOptions are the read-side options accepted by Browse.
type HubBrowseOptions struct {
	Page     int
	PageSize int
}

// HubSearchResponse pairs the sorted, deduped result list with a typed
// evidence value describing degraded conditions. The shape is wire-stable so
// gateway and TUI slices can serialise it directly.
type HubSearchResponse struct {
	Results  []HubSearchResult `json:"results"`
	Evidence HubSearchEvidence `json:"evidence,omitempty"`
}

// HubBrowseResponse is the paginated read-only view used by gateway and TUI
// skills browsing. It deliberately carries only metadata and result summaries;
// install/edit/review state belongs to later mutating rows.
type HubBrowseResponse struct {
	Results    []HubSearchResult `json:"results"`
	Evidence   HubSearchEvidence `json:"evidence,omitempty"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	Total      int               `json:"total"`
	TotalPages int               `json:"total_pages"`
}

// PreferHermesIndexProvider mirrors Hermes' source-router preference for the
// centralized skills index. When the injected index provider has cached entries,
// callers should consult it first and skip duplicate remote API providers for
// this search pass. If the index is unavailable, malformed, or empty, the
// caller-supplied fallback provider list is returned unchanged with typed
// evidence when available.
func PreferHermesIndexProvider(ctx context.Context, index HubRegistryProvider, fallbacks []HubRegistryProvider) ([]HubRegistryProvider, HubSearchEvidence) {
	fallbackCopy := append([]HubRegistryProvider(nil), fallbacks...)
	ctx = nonNilContext(ctx)
	if index == nil {
		return fallbackCopy, ""
	}
	if err := ctx.Err(); err != nil {
		return fallbackCopy, ""
	}
	snap, err := index.Snapshot(ctx)
	if err != nil {
		evidence := registryErrorEvidence(err)
		return fallbackCopy, evidence
	}
	if len(snap) == 0 {
		return fallbackCopy, HubSearchEvidenceNoResults
	}
	return []HubRegistryProvider{index}, ""
}

func registryErrorEvidence(err error) HubSearchEvidence {
	switch {
	case errors.Is(err, ErrRegistryUnavailable):
		return HubSearchEvidenceRegistryUnavailable
	case errors.Is(err, ErrRegistryRateLimited):
		return HubSearchEvidenceRateLimited
	case errors.Is(err, ErrRegistryMalformed):
		return HubSearchEvidenceRegistryMalformed
	default:
		return ""
	}
}

// Search merges the read-only snapshots from each provider, filters them by
// substring match on Name+Description, dedupes by InstallID, and sorts by
// Score descending then Name ascending. It never touches the active or
// inactive skill stores and never opens a network connection — providers are
// the only seam to live data.
func Search(ctx context.Context, query string, providers []HubRegistryProvider, opts HubSearchOptions) (HubSearchResponse, error) {
	ctx = nonNilContext(ctx)
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return HubSearchResponse{Evidence: HubSearchEvidenceEmptyQuery}, nil
	}

	needle := strings.ToLower(trimmed)
	results, evidence, err := collectHubResults(ctx, providers, func(r HubSearchResult) bool {
		haystack := strings.ToLower(r.Name + " " + r.Description)
		return strings.Contains(haystack, needle)
	})
	if err != nil {
		return HubSearchResponse{}, err
	}
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}
	return HubSearchResponse{Results: results, Evidence: evidence}, nil
}

// Browse returns a page of all read-only registry results. It shares Search's
// provider, dedupe, and sort rules, but does not require a query.
func Browse(ctx context.Context, providers []HubRegistryProvider, opts HubBrowseOptions) (HubBrowseResponse, error) {
	ctx = nonNilContext(ctx)
	results, evidence, err := collectHubResults(ctx, providers, func(HubSearchResult) bool { return true })
	if err != nil {
		return HubBrowseResponse{}, err
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	total := len(results)
	totalPages := 0
	if total > 0 {
		totalPages = ((total - 1) / pageSize) + 1
		if page > totalPages {
			page = totalPages
		}
	} else {
		page = 1
	}
	start := 0
	if page > 1 {
		start = (page - 1) * pageSize
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageResults := []HubSearchResult(nil)
	if start < end && start < total {
		pageResults = append(pageResults, results[start:end]...)
	}
	return HubBrowseResponse{Results: pageResults, Evidence: evidence, Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages}, nil
}

func collectHubResults(ctx context.Context, providers []HubRegistryProvider, keep func(HubSearchResult) bool) ([]HubSearchResult, HubSearchEvidence, error) {
	ctx = nonNilContext(ctx)
	var (
		merged      []HubSearchResult
		unavailable bool
		rateLimited bool
		malformed   bool
	)
	for _, p := range providers {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		if p == nil {
			continue
		}
		snap, err := p.Snapshot(ctx)
		if err != nil {
			switch {
			case errors.Is(err, ErrRegistryUnavailable):
				unavailable = true
			case errors.Is(err, ErrRegistryRateLimited):
				rateLimited = true
			case errors.Is(err, ErrRegistryMalformed):
				malformed = true
			default:
				return nil, "", err
			}
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		for _, r := range snap {
			if keep == nil || keep(r) {
				merged = append(merged, r)
			}
		}
	}

	deduped := make([]HubSearchResult, 0, len(merged))
	indexByInstallID := make(map[string]int, len(merged))
	for _, r := range merged {
		if r.InstallID == "" {
			deduped = append(deduped, r)
			continue
		}
		if i, ok := indexByInstallID[r.InstallID]; ok {
			if r.Score > deduped[i].Score {
				deduped[i] = r
			}
			continue
		}
		indexByInstallID[r.InstallID] = len(deduped)
		deduped = append(deduped, r)
	}

	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].Score != deduped[j].Score {
			return deduped[i].Score > deduped[j].Score
		}
		return deduped[i].Name < deduped[j].Name
	})

	var evidence HubSearchEvidence
	switch {
	case unavailable:
		evidence = HubSearchEvidenceRegistryUnavailable
	case rateLimited:
		evidence = HubSearchEvidenceRateLimited
	case malformed:
		evidence = HubSearchEvidenceRegistryMalformed
	case len(deduped) == 0:
		evidence = HubSearchEvidenceNoResults
	}
	return deduped, evidence, nil
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
