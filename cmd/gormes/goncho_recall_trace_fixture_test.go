package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service"
)

func writeStableGonchoRecallTraceFixture(t *testing.T) string {
	t.Helper()

	raw, err := stableGonchoRecallTraceFixture().StableJSON()
	if err != nil {
		t.Fatalf("marshal stable recall trace fixture: %v", err)
	}

	tracePath := filepath.Join(t.TempDir(), "stable_trace.golden.json")
	if err := os.WriteFile(tracePath, raw, 0o644); err != nil {
		t.Fatalf("write stable recall trace fixture: %v", err)
	}
	return tracePath
}

func stableGonchoRecallTraceFixture() goncho.RecallTrace {
	createdAt := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	authAt := time.Date(2026, 5, 17, 11, 0, 0, 0, time.UTC)
	rateAt := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	dbAt := time.Date(2026, 5, 17, 11, 30, 0, 0, time.UTC)

	auth := goncho.ScoredRecallCandidate{
		Candidate: goncho.RecallCandidate{
			MemoryID:   "mem-auth",
			SourceType: "conclusion",
			Content:    "JWT auth uses jose middleware.",
			SessionID:  "sess-auth",
			AgentID:    "gormes",
			ScopeID:    "team",
			CreatedAt:  authAt,
			Importance: 0.9,
			Provenance: []goncho.EvidenceItem{
				{Kind: "keyword", Note: "matched auth", Score: 0.8},
				{Kind: "semantic", Note: "embedding neighbor", Score: 0.9},
				{Kind: "graph", Note: "AUTH_USES edge", Score: 0.5},
				{Kind: "scope", Note: "same scope", Score: 1},
			},
		},
		Score: goncho.RecallScore{
			KeywordScore:     0.8,
			SemanticScore:    0.9,
			GraphScore:       0.5,
			RecencyScore:     0.999038,
			ImportanceScore:  0.9,
			ScopeScore:       1,
			RRFScore:         0.016314,
			DiversityPenalty: 0,
			FinalScore:       0.826218,
			WhySelected:      []string{"final_score=0.826218", "scoring_config=test-v1"},
		},
	}
	rate := goncho.ScoredRecallCandidate{
		Candidate: goncho.RecallCandidate{
			MemoryID:   "mem-rate",
			SourceType: "turn",
			Content:    "Rate limiting uses token bucket middleware.",
			SessionID:  "sess-auth",
			AgentID:    "gormes",
			ScopeID:    "team",
			CreatedAt:  rateAt,
			Importance: 0.7,
			Provenance: []goncho.EvidenceItem{
				{Kind: "keyword", Note: "matched rate limit", Score: 0.7},
				{Kind: "semantic", Note: "embedding neighbor", Score: 0.86},
				{Kind: "graph", Note: "related middleware edge", Score: 0.3},
				{Kind: "scope", Note: "same scope", Score: 1},
			},
		},
		Score: goncho.RecallScore{
			KeywordScore:     0.7,
			SemanticScore:    0.86,
			GraphScore:       0.3,
			RecencyScore:     0.998076,
			ImportanceScore:  0.7,
			ScopeScore:       1,
			RRFScore:         0.016039,
			DiversityPenalty: 0,
			FinalScore:       0.728847,
			WhySelected:      []string{"final_score=0.728847", "scoring_config=test-v1"},
		},
	}
	db := goncho.ScoredRecallCandidate{
		Candidate: goncho.RecallCandidate{
			MemoryID:   "mem-db",
			SourceType: "conclusion",
			Content:    "Database performance work found an N+1 query.",
			SessionID:  "sess-db",
			AgentID:    "gormes",
			ScopeID:    "team",
			CreatedAt:  dbAt,
			Importance: 0.6,
			Provenance: []goncho.EvidenceItem{
				{Kind: "keyword", Note: "weak lexical overlap", Score: 0.2},
				{Kind: "semantic", Note: "weak embedding neighbor", Score: 0.3},
				{Kind: "graph", Note: "database performance edge", Score: 0.95},
				{Kind: "scope", Note: "same scope", Score: 1},
			},
		},
		Score: goncho.RecallScore{
			KeywordScore:     0.2,
			SemanticScore:    0.3,
			GraphScore:       0.95,
			RecencyScore:     0.999519,
			ImportanceScore:  0.6,
			ScopeScore:       1,
			RRFScore:         0.016042,
			DiversityPenalty: 0,
			FinalScore:       0.555994,
			WhySelected: []string{
				"final_score=0.555994",
				"scoring_config=test-v1",
				"diversity_penalty=0.000000",
			},
		},
	}
	selectedAuth := auth
	selectedAuth.Score.WhySelected = []string{
		"final_score=0.826218",
		"scoring_config=test-v1",
		"diversity_penalty=0.000000",
	}

	rejectedRate := goncho.RejectedRecallCandidate{
		Candidate: rate.Candidate,
		Score: goncho.RecallScore{
			KeywordScore:     0.7,
			SemanticScore:    0.86,
			GraphScore:       0.3,
			RecencyScore:     0.998076,
			ImportanceScore:  0.7,
			ScopeScore:       1,
			RRFScore:         0.016039,
			DiversityPenalty: 0.3,
			FinalScore:       0.428847,
			WhySelected:      []string{"final_score=0.728847", "scoring_config=test-v1"},
		},
		Reason:      goncho.RecallRejectNotSelected,
		WhyRejected: []string{"limit=2"},
	}

	return goncho.RecallTrace{
		TraceID:         "b3765ad87524b8be6fcdf19dd43e946a8116dd8995e0e998f4a1f08bec69b9ef",
		PipelineVersion: "test-pipeline",
		CreatedAt:       createdAt,
		Query: goncho.RecallQuery{
			WorkspaceID: "default",
			Peer:        "user-juan",
			Query:       "auth rate limit",
			SessionKey:  "sess-auth",
			ScopeID:     "team",
			Limit:       2,
		},
		ScoringConfig: goncho.RecallScoringConfig{
			Version: "test-v1",
			Weights: map[string]float64{
				"keyword":    0.25,
				"semantic":   0.3,
				"graph":      0.2,
				"recency":    0.1,
				"importance": 0.1,
				"scope":      0.05,
			},
			RRFK:          60,
			MMRLambda:     0.7,
			DiversityKeys: []string{"session_id"},
			TokenBudget:   80,
		},
		Candidates: []goncho.ScoredRecallCandidate{auth, rate, db},
		Selected:   []goncho.ScoredRecallCandidate{selectedAuth, db},
		Rejected:   []goncho.RejectedRecallCandidate{rejectedRate},
		Warnings:   []goncho.RecallWarning{},
	}
}
