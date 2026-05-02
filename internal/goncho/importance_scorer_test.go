package goncho

import (
	"testing"
	"time"
)

func TestImportanceScorer_Score(t *testing.T) {
	s := NewImportanceScorer()
	now := time.Now()

	entry := MemoryToolEntry{
		ID: "mem_1", Content: "test", Importance: 0.8, CreatedAt: now,
	}
	score := s.Score(entry, 0.5, now)
	if score <= 0 || score > 1 {
		t.Fatalf("score = %f, want 0 < score <= 1", score)
	}
}

func TestImportanceScorer_Decay(t *testing.T) {
	s := NewImportanceScorer()
	now := time.Now()
	old := now.Add(-60 * 24 * time.Hour) // 60 days ago

	recent := MemoryToolEntry{ID: "r", Content: "recent", Importance: 0.5, CreatedAt: now}
	stale := MemoryToolEntry{ID: "s", Content: "stale", Importance: 0.5, CreatedAt: old}

	recentScore := s.Score(recent, 0.5, now)
	staleScore := s.Score(stale, 0.5, now)

	if staleScore >= recentScore {
		t.Fatalf("stale score %f >= recent score %f, want decay to reduce old entry score", staleScore, recentScore)
	}
}

func TestImportanceScorer_HighImportance(t *testing.T) {
	s := NewImportanceScorer()
	now := time.Now()

	low := MemoryToolEntry{ID: "l", Content: "low", Importance: 0.1, CreatedAt: now}
	high := MemoryToolEntry{ID: "h", Content: "high", Importance: 0.9, CreatedAt: now}

	if s.Score(high, 0.5, now) <= s.Score(low, 0.5, now) {
		t.Fatal("high importance entry should score higher than low importance")
	}
}

func TestDefaultDecayCurve_HalfLife(t *testing.T) {
	now := time.Now()
	oneHalfLife := now.Add(-30 * 24 * time.Hour)

	val := DefaultDecayCurve(oneHalfLife, now)
	if val < 0.45 || val > 0.55 {
		t.Fatalf("after one half-life, decay = %f, want ~0.5", val)
	}
}

func TestEffectiveImportance(t *testing.T) {
	s := NewImportanceScorer()
	now := time.Now()

	entry := MemoryToolEntry{ID: "e", Content: "test", Importance: 0.8, CreatedAt: now}
	eff := s.EffectiveImportance(entry, now)

	if eff < 0.79 {
		t.Fatalf("effective importance = %f for fresh entry with 0.8 importance, want ~0.8", eff)
	}
}
