package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/selection"

// HybridSelector combines lexical and semantic scoring for skill selection.
type HybridSelector = selection.HybridSelector

// SemanticEmbedder returns an embedding for a text string, used for
// semantic similarity scoring. Nil means lexical-only selection.
type SemanticEmbedder = selection.SemanticEmbedder

// ScoredSkill is a skill with its selection evidence.
type ScoredSkill = selection.ScoredSkill
