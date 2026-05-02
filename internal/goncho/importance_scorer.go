package goncho

import (
	"math"
	"time"
)

const defaultDecayHalfLife = 30 * 24 * time.Hour // 30 days

type ImportanceScorer struct {
	alphaRecency   float64
	betaImportance float64
	gammaRelevance float64
	decayHalfLife  time.Duration
}

func NewImportanceScorer() *ImportanceScorer {
	return &ImportanceScorer{
		alphaRecency:   0.3,
		betaImportance: 0.5,
		gammaRelevance: 0.2,
		decayHalfLife:  defaultDecayHalfLife,
	}
}

func (s *ImportanceScorer) Score(entry MemoryToolEntry, relevanceScore float64, now time.Time) float64 {
	recency := s.recencyScore(entry.CreatedAt, now)
	score := s.alphaRecency*recency + s.betaImportance*entry.Importance + s.gammaRelevance*relevanceScore
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}

func (s *ImportanceScorer) recencyScore(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}
	halfLives := float64(age) / float64(s.decayHalfLife)
	return math.Exp2(-halfLives)
}

func (s *ImportanceScorer) EffectiveImportance(entry MemoryToolEntry, now time.Time) float64 {
	base := entry.Importance * s.recencyScore(entry.CreatedAt, now)
	if base < 0.01 {
		base = 0.01
	}
	return base
}

func DefaultDecayCurve(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}
	halfLives := float64(age) / float64(defaultDecayHalfLife)
	return math.Exp2(-halfLives)
}
