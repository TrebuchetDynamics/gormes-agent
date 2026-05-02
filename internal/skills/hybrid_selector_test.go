package skills

import (
	"sort"
	"testing"
)

type nilEmbedder struct{}

func (n nilEmbedder) Embed(ctx interface{}, text string) ([]float32, error) {
	return nil, nil
}

func TestHybridSelector_LexicalOnly(t *testing.T) {
	s := &HybridSelector{Embedder: nil}
	skills := []Skill{
		{Name: "python-debugging", Description: "Debug Python code", Body: "use pdb", ReviewState: "reviewed", HermesTags: []string{"python"}},
		{Name: "web-scraping", Description: "Scrape websites", Body: "use beautifulsoup", ReviewState: "reviewed"},
		{Name: "git-workflow", Description: "Git workflow", Body: "conventional commits", ReviewState: "verified"},
	}
	results := s.Select(skills, "python debug", 2)
	if len(results) != 2 {
		t.Fatalf("Select = %d, want 2", len(results))
	}
	if results[0].Skill.Name != "python-debugging" {
		t.Errorf("first = %s, want python-debugging", results[0].Skill.Name)
	}
	if !results[0].Degraded {
		t.Error("expected degraded flag when embedder is nil")
	}
}

func TestHybridSelector_FiltersDisabledAndInvalid(t *testing.T) {
	s := &HybridSelector{}
	skills := []Skill{
		{Name: "valid-skill", Description: "A valid skill", ReviewState: "reviewed"},
		{Name: "disabled-skill", Description: "Should be filtered", ReviewState: "unreviewed"},
	}
	results := s.Select(skills, "skill", 5)
	if len(results) != 1 {
		t.Fatalf("Select = %d, want 1 (only valid skill)", len(results))
	}
	if results[0].Skill.Name != "valid-skill" {
		t.Errorf("selected = %s, want valid-skill", results[0].Skill.Name)
	}
}

func TestHybridSelector_SourceAwareRanking(t *testing.T) {
	s := &HybridSelector{}
	skills := []Skill{
		{Name: "a-skill", Description: "test skill a", ReviewState: "reviewed"},
		{Name: "b-skill", Description: "test skill b", ReviewState: "imported"},
	}
	results := s.Select(skills, "test skill", 5)
	if len(results) < 2 {
		t.Fatalf("Select = %d, want at least 2", len(results))
	}
	// Curated source should outrank imported
	if results[0].Skill.Name != "a-skill" {
		t.Errorf("first = %s, want a-skill (curated)", results[0].Skill.Name)
	}
}

func TestHybridSelector_EmptySkills(t *testing.T) {
	s := &HybridSelector{}
	if s.Select(nil, "query", 5) != nil {
		t.Error("expected nil for empty skills")
	}
	if s.Select([]Skill{}, "query", 5) != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestHybridSelector_ScoreExplanation(t *testing.T) {
	s := &HybridSelector{Embedder: nil}
	skills := []Skill{
		{Name: "test-skill", Description: "A test skill for testing", ReviewState: "reviewed"},
	}
	results := s.Select(skills, "test", 1)
	if len(results) == 0 {
		t.Fatal("expected result")
	}
	ev := results[0].Evidence()
	if ev == "" {
		t.Error("expected non-empty evidence string")
	}
	if results[0].LexicalScore == 0 {
		t.Error("expected positive lexical score for matching query")
	}
}

func TestHybridSelector_SortsByTotalScore(t *testing.T) {
	s := &HybridSelector{}
	skills := []Skill{
		{Name: "exact-hit", Description: "python debugging tool", ReviewState: "reviewed"},
		{Name: "partial-hit", Description: "general programming", ReviewState: "reviewed"},
		{Name: "no-match", Description: "something else entirely", ReviewState: "imported"},
	}
	results := s.Select(skills, "python debug", 3)
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Skill.Name
	}
	sort.Strings(names)
	if !containsString([]string{"exact-hit", "partial-hit", "no-match"}, names[0]) {
		t.Errorf("unexpected result order: %v", names)
	}
}

func containsString(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
