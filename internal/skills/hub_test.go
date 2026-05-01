package skills

import (
	"errors"
	"testing"
)

func TestHubRegistry_UnifiedSearch(t *testing.T) {
	p1 := &collectingProvider{
		sourceID:   "source-a",
		trustLevel: "trusted",
		results: []SkillMeta{
			{Name: "alpha", Description: "alpha skill", Source: "source-a", Identifier: "a/alpha", Trust: "trusted"},
			{Name: "shared-one", Description: "shared from p1", Source: "source-a", Identifier: "shared/dup", Trust: "trusted"},
			{Name: "zeta", Description: "zeta skill", Source: "source-a", Identifier: "a/zeta", Trust: "trusted"},
		},
	}
	p2 := &collectingProvider{
		sourceID:   "source-b",
		trustLevel: "community",
		results: []SkillMeta{
			{Name: "beta", Description: "beta skill", Source: "source-b", Identifier: "b/beta", Trust: "community"},
			{Name: "shared-one", Description: "shared from p2", Source: "source-b", Identifier: "shared/dup", Trust: "community"},
			{Name: "mu", Description: "mu skill", Source: "source-b", Identifier: "b/mu", Trust: "community"},
		},
	}

	registry := NewHubRegistry(p1, p2)
	got, err := registry.UnifiedSearch("skill", 10)
	if err != nil {
		t.Fatalf("UnifiedSearch returned unexpected error: %v", err)
	}

	wantNames := []string{"alpha", "beta", "mu", "shared-one", "zeta"}
	if len(got) != len(wantNames) {
		t.Fatalf("result count = %d, want %d", len(got), len(wantNames))
	}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Errorf("Results[%d].Name = %q, want %q", i, got[i].Name, want)
		}
	}

	dupCount := 0
	for _, r := range got {
		if r.Identifier == "shared/dup" {
			dupCount++
		}
	}
	if dupCount != 1 {
		t.Errorf("shared/dup appeared %d times, want exactly 1 deduplication", dupCount)
	}
}

func TestHubRegistry_EmptyProviders(t *testing.T) {
	registry := NewHubRegistry()
	got, err := registry.UnifiedSearch("anything", 5)
	if err != nil {
		t.Fatalf("UnifiedSearch returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestHubRegistry_NilProviderSkipped(t *testing.T) {
	good := &collectingProvider{
		sourceID:   "good",
		trustLevel: "community",
		results: []SkillMeta{
			{Name: "test", Description: "test skill", Source: "good", Identifier: "good/test", Trust: "community"},
		},
	}
	registry := NewHubRegistry(nil, good, nil)
	got, err := registry.UnifiedSearch("test", 5)
	if err != nil {
		t.Fatalf("UnifiedSearch returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("result count = %d, want 1", len(got))
	}
}

func TestHubRegistry_EmptyQueryReturnsNil(t *testing.T) {
	registry := NewHubRegistry(&collectingProvider{
		sourceID:   "test",
		trustLevel: "community",
		results: []SkillMeta{{Name: "should-not-appear"}},
	})
	for _, query := range []string{"", "   ", "\t\n  "} {
		got, err := registry.UnifiedSearch(query, 5)
		if err != nil {
			t.Fatalf("UnifiedSearch(%q) returned unexpected error: %v", query, err)
		}
		if got != nil {
			t.Errorf("UnifiedSearch(%q) = %v, want nil", query, got)
		}
	}
}

func TestHubRegistry_ProviderErrorSwallowedOnPartialSuccess(t *testing.T) {
	failing := &collectingProvider{
		sourceID:   "failing",
		trustLevel: "community",
		err:       errors.New("provider error"),
	}
	healthy := &collectingProvider{
		sourceID:   "healthy",
		trustLevel: "trusted",
		results: []SkillMeta{
			{Name: "ok", Description: "works", Source: "healthy", Identifier: "h/ok", Trust: "trusted"},
		},
	}
	registry := NewHubRegistry(failing, healthy)
	got, err := registry.UnifiedSearch("ok", 5)
	if err != nil {
		t.Fatalf("expected no error when at least one provider succeeds, got: %v", err)
	}
	if len(got) != 1 || got[0].Name != "ok" {
		t.Errorf("unexpected results: %v", got)
	}
}

func TestHubRegistry_ProviderErrorReturnedOnNoSuccess(t *testing.T) {
	failing := &collectingProvider{
		sourceID:   "failing",
		trustLevel: "community",
		err:       errors.New("provider error"),
	}
	registry := NewHubRegistry(failing)
	got, err := registry.UnifiedSearch("test", 5)
	if err == nil {
		t.Fatalf("expected error when all providers fail, got nil")
	}
	if got != nil {
		t.Errorf("expected nil results on error, got %v", got)
	}
}

func TestHubRegistry_LimitApplied(t *testing.T) {
	registry := NewHubRegistry(&collectingProvider{
		sourceID:   "test",
		trustLevel: "community",
		results: []SkillMeta{
			{Name: "a", Description: "a skill", Source: "test", Identifier: "t/a"},
			{Name: "b", Description: "b skill", Source: "test", Identifier: "t/b"},
			{Name: "c", Description: "c skill", Source: "test", Identifier: "t/c"},
		},
	})
	got, err := registry.UnifiedSearch("skill", 2)
	if err != nil {
		t.Fatalf("UnifiedSearch returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("result count = %d, want 2", len(got))
	}
}

func TestSkillsShProvider_StubReturnsNil(t *testing.T) {
	got, err := SkillsShProvider.Search("test", 5)
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil from stub, got %v", got)
	}
	if SkillsShProvider.SourceID() != "skills-sh" {
		t.Errorf("SourceID = %q, want %q", SkillsShProvider.SourceID(), "skills-sh")
	}
	if SkillsShProvider.TrustLevel() != "community" {
		t.Errorf("TrustLevel = %q, want %q", SkillsShProvider.TrustLevel(), "community")
	}
}

type collectingProvider struct {
	sourceID   string
	trustLevel string
	results    []SkillMeta
	err        error
}

func (p *collectingProvider) SourceID() string { return p.sourceID }

func (p *collectingProvider) Search(_ string, _ int) ([]SkillMeta, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.results, nil
}

func (p *collectingProvider) TrustLevel() string { return p.trustLevel }
