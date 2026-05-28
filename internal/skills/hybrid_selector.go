package skills

import (
	"fmt"
	"sort"
	"strings"
)

// HybridSelector combines lexical and semantic scoring for skill selection.
type HybridSelector struct {
	Embedder SemanticEmbedder
}

// SemanticEmbedder returns an embedding for a text string, used for
// semantic similarity scoring. Nil means lexical-only selection.
type SemanticEmbedder interface {
	Embed(ctx interface{}, text string) ([]float32, error)
}

// ScoredSkill is a skill with its selection evidence.
type ScoredSkill struct {
	Skill         Skill   `json:"skill"`
	LexicalScore  int     `json:"lexical_score"`
	SemanticScore float64 `json:"semantic_score"`
	TotalScore    float64 `json:"total_score"`
	SourceBoost   float64 `json:"source_boost,omitempty"`
	DampingFactor float64 `json:"damping_factor,omitempty"`
	Excluded      bool    `json:"excluded"`
	ExcludeReason string  `json:"exclude_reason,omitempty"`
	Degraded      bool    `json:"degraded,omitempty"`
	DegradeReason string  `json:"degrade_reason,omitempty"`
}

// Select applies hybrid scoring and returns the top N skills.
func (s *HybridSelector) Select(skills []Skill, query string, max int) []ScoredSkill {
	if len(skills) == 0 {
		return nil
	}
	if max <= 0 {
		max = DefaultSelectionCap
	}

	candidates := s.filterCandidates(skills)
	tokens := tokenize(query)

	scored := make([]ScoredSkill, 0, len(candidates))
	for _, skill := range candidates {
		ss := s.score(skill, tokens, query)
		if ss.Excluded {
			continue
		}
		scored = append(scored, ss)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].TotalScore != scored[j].TotalScore {
			return scored[i].TotalScore > scored[j].TotalScore
		}
		return scored[i].Skill.Name < scored[j].Skill.Name
	})

	if len(scored) > max {
		scored = scored[:max]
	}
	return scored
}

// SelectNames returns just the skill names for backward compatibility.
func (s *HybridSelector) SelectNames(skills []Skill, query string, max int) []string {
	selected := s.Select(skills, query, max)
	names := make([]string, len(selected))
	for i, ss := range selected {
		names[i] = ss.Skill.Name
	}
	return names
}

func (s *HybridSelector) filterCandidates(skills []Skill) []Skill {
	out := make([]Skill, 0, len(skills))
	for _, sk := range skills {
		if sk.ReviewState == "unreviewed" || sk.ReviewState == "draft" {
			continue
		}
		out = append(out, sk)
	}
	return out
}

func (s *HybridSelector) score(sk Skill, tokens []string, query string) ScoredSkill {
	lexical := scoreSkill(sk, tokens)
	ss := ScoredSkill{
		Skill:        sk,
		LexicalScore: lexical,
	}

	// Semantic scoring
	if s.Embedder != nil {
		_, err := s.Embedder.Embed(nil, query)
		if err != nil {
			ss.Degraded = true
			ss.DegradeReason = "semantic embedding failed: " + err.Error()
		} else {
			ss.Degraded = true
			ss.DegradeReason = "skill embedding storage not yet implemented"
		}
	} else {
		ss.Degraded = true
		ss.DegradeReason = "semantic embedder not configured"
	}

	// Source-aware damping
	ss.SourceBoost, ss.DampingFactor = sourceAdjustment(sk)

	// Total score: lexical base + semantic bonus + source boost, multiplied by damping
	lexBase := float64(lexical) / 100.0
	semBonus := ss.SemanticScore * 50 // semantic score weight
	ss.TotalScore = (lexBase + ss.SourceBoost + semBonus) * (1.0 - ss.DampingFactor)
	if ss.TotalScore < 0 {
		ss.TotalScore = 0
	}

	return ss
}

func sourceAdjustment(sk Skill) (boost float64, damping float64) {
	switch strings.ToLower(sk.ReviewState) {
	case "reviewed", "verified", "approved":
		return 10.0, 0.0
	case "imported", "raw":
		return 0.0, 0.5
	case "unreviewed", "draft":
		return 0.0, 0.3
	default:
		return 0.0, 0.0
	}
}

// cosineSimilarity returns the cosine similarity between two float32 vectors.
// SkillSelectionEvidence produces a human-readable explanation.
func (ss ScoredSkill) Evidence() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("lexical=%d", ss.LexicalScore))
	if ss.SemanticScore > 0 {
		parts = append(parts, fmt.Sprintf("semantic=%.3f", ss.SemanticScore))
	}
	if ss.SourceBoost > 0 {
		parts = append(parts, fmt.Sprintf("source_boost=%.1f", ss.SourceBoost))
	}
	if ss.DampingFactor > 0 {
		parts = append(parts, fmt.Sprintf("damping=%.2f", ss.DampingFactor))
	}
	if ss.Degraded {
		parts = append(parts, "degraded="+ss.DegradeReason)
	}
	return strings.Join(parts, " ")
}
