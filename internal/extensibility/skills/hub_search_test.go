package skills

import (
	"context"
	"testing"
)

type recordingHubRegistryProvider struct {
	called bool
}

func (p *recordingHubRegistryProvider) Snapshot(context.Context) ([]HubSearchResult, error) {
	p.called = true
	return []HubSearchResult{{Name: "alpha", Description: "alpha skill", InstallID: "skills/alpha", Score: 0.9}}, nil
}

type cancelingHubRegistryProvider struct {
	cancel context.CancelFunc
}

func (p cancelingHubRegistryProvider) Snapshot(context.Context) ([]HubSearchResult, error) {
	p.cancel()
	return []HubSearchResult{{Name: "alpha", Description: "alpha skill", InstallID: "skills/alpha", Score: 0.9}}, nil
}

func TestHubSearchContextCanceledDuringProviderDoesNotReturnResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := cancelingHubRegistryProvider{cancel: cancel}

	resp, err := Search(ctx, "alpha", []HubRegistryProvider{provider}, HubSearchOptions{})
	if err != context.Canceled {
		t.Fatalf("Search err = %v response=%+v, want context.Canceled", err, resp)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("Search returned results after cancellation: %+v", resp.Results)
	}
}

func TestPreferHermesIndexProviderCanceledContextDoesNotReadIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	index := &recordingHubRegistryProvider{}
	fallback := NewInMemoryRegistryProvider([]HubSearchResult{{Name: "fallback", InstallID: "skills/fallback"}}, nil)

	providers, evidence := PreferHermesIndexProvider(ctx, index, []HubRegistryProvider{fallback})
	if evidence != "" {
		t.Fatalf("evidence = %q, want empty when cancellation leaves fallback routing unchanged", evidence)
	}
	if len(providers) != 1 || providers[0] != fallback {
		t.Fatalf("providers = %#v, want fallback provider unchanged", providers)
	}
	if index.called {
		t.Fatal("PreferHermesIndexProvider called index provider after context was canceled")
	}
}

func TestHubSearchCanceledContextDoesNotReadProviders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := &recordingHubRegistryProvider{}

	resp, err := Search(ctx, "alpha", []HubRegistryProvider{provider}, HubSearchOptions{})
	if err != context.Canceled {
		t.Fatalf("Search canceled err = %v response=%+v, want context.Canceled", err, resp)
	}
	if provider.called {
		t.Fatal("Search called provider after context was canceled")
	}

	provider = &recordingHubRegistryProvider{}
	browseResp, err := Browse(ctx, []HubRegistryProvider{provider}, HubBrowseOptions{})
	if err != context.Canceled {
		t.Fatalf("Browse canceled err = %v response=%+v, want context.Canceled", err, browseResp)
	}
	if provider.called {
		t.Fatal("Browse called provider after context was canceled")
	}
}

// TestHubSearchEmptyQueryReturnsEmptyEvidence asserts that an empty or
// whitespace-only query short-circuits before touching providers and reports
// HubSearchEvidenceEmptyQuery instead of forwarding any results.
func TestHubSearchEmptyQueryReturnsEmptyEvidence(t *testing.T) {
	provider := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha", Description: "alpha skill", Source: "fixture", InstallID: "fixture/alpha", Score: 0.50},
	}, nil)

	for _, query := range []string{"", "   ", "\t\n  "} {
		resp, err := Search(context.Background(), query, []HubRegistryProvider{provider}, HubSearchOptions{})
		if err != nil {
			t.Fatalf("Search(query=%q) returned unexpected error: %v", query, err)
		}
		if resp.Evidence != HubSearchEvidenceEmptyQuery {
			t.Errorf("Search(query=%q) Evidence = %q, want %q", query, resp.Evidence, HubSearchEvidenceEmptyQuery)
		}
		if len(resp.Results) != 0 {
			t.Errorf("Search(query=%q) Results = %v, want empty", query, resp.Results)
		}
	}
}

// TestHubSearchSortsAndDedupes asserts that results from multiple providers
// are merged, deduped by InstallID, and sorted by Score descending then Name
// ascending.
func TestHubSearchSortsAndDedupes(t *testing.T) {
	p1 := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha", Description: "alpha skill", Source: "p1", InstallID: "p/alpha", Score: 0.30},
		{Name: "duplicate", Description: "shared skill", Source: "p1", InstallID: "shared/dup", Score: 0.50},
		{Name: "zeta", Description: "zeta skill", Source: "p1", InstallID: "p/zeta", Score: 0.80},
	}, nil)
	p2 := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha2", Description: "alpha2 skill", Source: "p2", InstallID: "p/alpha2", Score: 0.30},
		{Name: "duplicate", Description: "shared skill", Source: "p2", InstallID: "shared/dup", Score: 0.50},
		{Name: "mu", Description: "mu skill", Source: "p2", InstallID: "p/mu", Score: 0.50},
	}, nil)

	resp, err := Search(context.Background(), "skill", []HubRegistryProvider{p1, p2}, HubSearchOptions{})
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if resp.Evidence == HubSearchEvidenceEmptyQuery ||
		resp.Evidence == HubSearchEvidenceRegistryUnavailable ||
		resp.Evidence == HubSearchEvidenceRateLimited ||
		resp.Evidence == HubSearchEvidenceNoResults {
		t.Errorf("Evidence = %q, want a non-degraded value", resp.Evidence)
	}

	wantOrder := []string{"zeta", "duplicate", "mu", "alpha", "alpha2"}
	gotOrder := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		gotOrder = append(gotOrder, r.Name)
	}
	if len(gotOrder) != len(wantOrder) {
		t.Fatalf("Results length = %d (%v), want %d (%v)", len(gotOrder), gotOrder, len(wantOrder), wantOrder)
	}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Errorf("Results[%d].Name = %q, want %q (full=%v)", i, gotOrder[i], wantOrder[i], gotOrder)
		}
	}

	dupCount := 0
	for _, r := range resp.Results {
		if r.InstallID == "shared/dup" {
			dupCount++
		}
	}
	if dupCount != 1 {
		t.Errorf("shared/dup appeared %d times, want 1", dupCount)
	}
}

// TestHubSearchRegistryUnavailable asserts that Search keeps results from
// healthy providers when one provider returns ErrRegistryUnavailable, while
// still surfacing the degraded condition via Evidence.
func TestHubSearchRegistryUnavailable(t *testing.T) {
	failed := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "should-not-appear", InstallID: "broken/x", Score: 1.0},
	}, ErrRegistryUnavailable)
	healthy := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha", Description: "alpha skill", Source: "ok", InstallID: "ok/alpha", Score: 0.50},
	}, nil)

	resp, err := Search(context.Background(), "alpha", []HubRegistryProvider{failed, healthy}, HubSearchOptions{})
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if resp.Evidence != HubSearchEvidenceRegistryUnavailable {
		t.Errorf("Evidence = %q, want %q", resp.Evidence, HubSearchEvidenceRegistryUnavailable)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("Results count = %d (%v), want 1", len(resp.Results), resp.Results)
	}
	if resp.Results[0].InstallID != "ok/alpha" {
		t.Errorf("Results[0].InstallID = %q, want %q", resp.Results[0].InstallID, "ok/alpha")
	}
}

// TestHubSearchNoResults asserts that a non-empty query with no provider
// matches reports HubSearchEvidenceNoResults instead of leaving callers to
// guess from an empty Results slice.
func TestHubSearchNoResults(t *testing.T) {
	provider := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha", Description: "alpha skill", Source: "fixture", InstallID: "fixture/alpha", Score: 0.50},
		{Name: "beta", Description: "beta skill", Source: "fixture", InstallID: "fixture/beta", Score: 0.40},
	}, nil)

	resp, err := Search(context.Background(), "no-such-skill-anywhere", []HubRegistryProvider{provider}, HubSearchOptions{})
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if resp.Evidence != HubSearchEvidenceNoResults {
		t.Errorf("Evidence = %q, want %q", resp.Evidence, HubSearchEvidenceNoResults)
	}
	if len(resp.Results) != 0 {
		t.Errorf("Results = %v, want empty", resp.Results)
	}
}

func TestHubSearchBrowsePagesReadOnlySnapshots(t *testing.T) {
	provider := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "zeta", Description: "zeta skill", Source: "hermes-index", InstallID: "skills/zeta", Score: 0.20, TrustLevel: "community"},
		{Name: "alpha", Description: "alpha skill", Source: "hermes-index", InstallID: "skills/alpha", Score: 0.90, TrustLevel: "trusted"},
		{Name: "alpha duplicate", Description: "duplicate should lose", Source: "remote", InstallID: "skills/alpha", Score: 0.10, TrustLevel: "community"},
	}, nil)

	resp, err := Browse(context.Background(), []HubRegistryProvider{provider}, HubBrowseOptions{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("Browse returned unexpected error: %v", err)
	}
	if resp.Page != 2 || resp.PageSize != 1 || resp.Total != 2 || resp.TotalPages != 2 {
		t.Fatalf("page metadata = page %d size %d total %d total_pages %d, want page 2 size 1 total 2 total_pages 2", resp.Page, resp.PageSize, resp.Total, resp.TotalPages)
	}
	if len(resp.Results) != 1 || resp.Results[0].Name != "zeta" {
		t.Fatalf("Results = %+v, want second sorted/deduped result zeta", resp.Results)
	}
	if resp.Evidence != "" {
		t.Fatalf("Evidence = %q, want empty for successful browse", resp.Evidence)
	}
}

func TestHubBrowseNoResultsClampsPageMetadata(t *testing.T) {
	resp, err := Browse(context.Background(), nil, HubBrowseOptions{Page: 99, PageSize: 10})
	if err != nil {
		t.Fatalf("Browse returned unexpected error: %v", err)
	}
	if resp.Page != 1 || resp.TotalPages != 0 || resp.Total != 0 || resp.Evidence != HubSearchEvidenceNoResults {
		t.Fatalf("page metadata = page %d total_pages %d total %d evidence %q, want page 1 total_pages 0 total 0 no_results", resp.Page, resp.TotalPages, resp.Total, resp.Evidence)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("Results = %+v, want empty", resp.Results)
	}
}

func TestHubBrowseHugePageSizeDoesNotOverflowPageMetadata(t *testing.T) {
	provider := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha", Description: "alpha skill", InstallID: "skills/alpha", Score: 0.9},
		{Name: "beta", Description: "beta skill", InstallID: "skills/beta", Score: 0.8},
	}, nil)

	resp, err := Browse(context.Background(), []HubRegistryProvider{provider}, HubBrowseOptions{Page: 1, PageSize: int(^uint(0) >> 1)})
	if err != nil {
		t.Fatalf("Browse returned unexpected error: %v", err)
	}
	if resp.Page != 1 || resp.TotalPages != 1 || resp.Total != 2 {
		t.Fatalf("page metadata = page %d total_pages %d total %d, want page 1/1 total 2", resp.Page, resp.TotalPages, resp.Total)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("Results count = %d, want all results with huge page size", len(resp.Results))
	}
}

func TestHubSearchRegistryMalformed(t *testing.T) {
	failed := NewInMemoryRegistryProvider(nil, ErrRegistryMalformed)
	healthy := NewInMemoryRegistryProvider([]HubSearchResult{
		{Name: "alpha", Description: "alpha skill", Source: "ok", InstallID: "ok/alpha", Score: 0.50},
	}, nil)

	resp, err := Search(context.Background(), "alpha", []HubRegistryProvider{failed, healthy}, HubSearchOptions{})
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if resp.Evidence != HubSearchEvidenceRegistryMalformed {
		t.Errorf("Evidence = %q, want %q", resp.Evidence, HubSearchEvidenceRegistryMalformed)
	}
	if len(resp.Results) != 1 || resp.Results[0].InstallID != "ok/alpha" {
		t.Fatalf("Results = %v, want ok/alpha from healthy provider", resp.Results)
	}
}
