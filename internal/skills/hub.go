package skills

import (
	"sort"
	"strings"
)

// SkillMeta is a registry-agnostic metadata record returned by HubProvider
// searches. It is the canonical shape for unified hub results before any
// gateway-specific augmentation (scores, tags, etc.) is applied.
type SkillMeta struct {
	Name        string
	Description string
	Category    string
	Source      string
	Identifier  string
	Trust       string
}

// HubProvider is the minimal interface a skill-hub source must implement to
// participate in UnifiedSearch. It mirrors the "SkillSource" ABC contract from
// Python's tools/skills_hub.py: sources are identified, queried, and graded by
// trust level.
type HubProvider interface {
	// SourceID returns the stable identifier for this provider, e.g.
	// "skills-sh", "hermes-index", "github". The value is used for dedup key
	// construction and diagnostics; it must not be empty.
	SourceID() string
	// Search performs a hub-native query and returns matched skill metadata.
	// Implementations must not mutate active or inactive skill stores.
	Search(query string, limit int) ([]SkillMeta, error)
	// TrustLevel returns the trust classification for results from this
	// provider: "trusted", "community", or "unverified".
	TrustLevel() string
}

// HubRegistry collects multiple HubProvider instances and exposes a unified
// search that merges, deduplicates, and returns combined results.
type HubRegistry struct {
	providers []HubProvider
}

// NewHubRegistry returns a registry holding the supplied providers. If no
// providers are supplied the registry still functions but returns empty
// results on UnifiedSearch.
func NewHubRegistry(providers ...HubProvider) *HubRegistry {
	return &HubRegistry{providers: providers}
}

// UnifiedSearch queries all registered providers, merges the result sets,
// deduplicates by Identifier, and returns up to `limit` records sorted by
// Name ascending. When no providers are registered or all return no results,
// the returned slice is empty and the error is nil. Provider errors are
// collected but do not abort the merge; the first non-nil error is returned
// only when no provider contributed any result.
func (h *HubRegistry) UnifiedSearch(query string, limit int) ([]SkillMeta, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, nil
	}

	var (
		merged  []SkillMeta
		errSeen error
	)

	for _, p := range h.providers {
		if p == nil {
			continue
		}
		meta, err := p.Search(trimmed, limit)
		if err != nil {
			if errSeen == nil {
				errSeen = err
			}
			continue
		}
		for _, m := range meta {
			m.Source = firstNonEmpty(m.Source, p.SourceID())
			m.Trust = firstNonEmpty(m.Trust, p.TrustLevel())
			merged = append(merged, m)
		}
	}

	if len(merged) == 0 && errSeen != nil {
		return nil, errSeen
	}

	deduped := deduplicateByIdentifier(merged)

	sort.SliceStable(deduped, func(i, j int) bool {
		return deduped[i].Name < deduped[j].Name
	})

	if limit > 0 && len(deduped) > limit {
		deduped = deduped[:limit]
	}

	return deduped, nil
}

// deduplicateByIdentifier keeps the first occurrence of each unique Identifier
// within the input slice. Entries with an empty Identifier are kept.
func deduplicateByIdentifier(in []SkillMeta) []SkillMeta {
	out := make([]SkillMeta, 0, len(in))
	seen := make(map[string]int, len(in))
	for _, m := range in {
		if m.Identifier == "" {
			out = append(out, m)
			continue
		}
		if _, ok := seen[m.Identifier]; !ok {
			seen[m.Identifier] = len(out)
			out = append(out, m)
		}
	}
	return out
}

// skillsShProvider is a stub HubProvider that implements the skills.sh source
// contract. It returns no results by default; replace with a live adapter that
// calls the skills.sh API in a later slice.
type skillsShProvider struct{}

// SourceID implements HubProvider.
func (s *skillsShProvider) SourceID() string { return "skills-sh" }

// Search implements HubProvider. The stub always returns an empty slice.
func (s *skillsShProvider) Search(query string, limit int) ([]SkillMeta, error) {
	return nil, nil
}

// TrustLevel implements HubProvider.
func (s *skillsShProvider) TrustLevel() string { return "community" }

// SkillsShProvider is the singleton instance of the skills.sh stub provider.
var SkillsShProvider HubProvider = &skillsShProvider{}
