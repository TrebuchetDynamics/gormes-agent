package goncho

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestGonchoProofMatrix_StorageRetrievalTraceAndOperatorEvidence(t *testing.T) {
	t.Run("storage_retrieval_context_and_workspace_boundaries", func(t *testing.T) {
		svc, cleanup := newTestService(t)
		defer cleanup()
		other := NewService(svc.db, Config{WorkspaceID: "other", ObserverPeerID: "gormes"}, nil)

		ctx := context.Background()
		if err := svc.SetProfile(ctx, "user-juan", []string{"Prefers evidence-first proof reports"}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateMessages(ctx, CreateMessagesParams{
			SessionKey: "sess-proof",
			Messages: []CreateMessage{{
				Peer:      "user-juan",
				Role:      "user",
				Content:   "Please prove Goncho with deterministic fixtures.",
				CreatedAt: time.Unix(1700000100, 0).UTC(),
			}},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Conclude(ctx, ConcludeParams{
			Peer:       "user-juan",
			Conclusion: "Goncho proof matrix codename orchid lives in default workspace.",
			SessionKey: "sess-proof",
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := other.Conclude(ctx, ConcludeParams{
			Peer:       "user-juan",
			Conclusion: "Goncho proof matrix codename orchid must not leak from other workspace.",
			SessionKey: "sess-proof",
		}); err != nil {
			t.Fatal(err)
		}

		search, err := svc.Search(ctx, SearchParams{
			Peer:       "user-juan",
			Query:      "codename orchid",
			SessionKey: "sess-proof",
			MaxTokens:  200,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(search.Results) != 1 || !strings.Contains(search.Results[0].Content, "default workspace") {
			t.Fatalf("search results = %+v, want default workspace proof conclusion", search.Results)
		}
		if strings.Contains(search.Results[0].Content, "other workspace") {
			t.Fatalf("search leaked cross-workspace content: %+v", search.Results)
		}

		contextResult, err := svc.Context(ctx, ContextParams{
			Peer:       "user-juan",
			Query:      "codename orchid",
			SessionKey: "sess-proof",
			MaxTokens:  400,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(contextResult.PeerCard) != 1 || len(contextResult.Conclusions) != 1 || len(contextResult.RecentMessages) != 1 {
			t.Fatalf("context = %+v, want peer card, one conclusion, one recent message", contextResult)
		}
		if contextResult.Representation == "" {
			t.Fatal("context representation is empty")
		}

		deleted, err := svc.DeleteSession(ctx, "sess-proof")
		if err != nil {
			t.Fatal(err)
		}
		if deleted.MessagesDeleted != 1 || deleted.ConclusionsDeleted != 1 {
			t.Fatalf("DeleteSession = %+v, want one message and one session conclusion deleted", deleted)
		}
		afterDelete, err := svc.Search(ctx, SearchParams{
			Peer:       "user-juan",
			Query:      "codename orchid",
			SessionKey: "sess-proof",
			MaxTokens:  200,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(afterDelete.Results) != 0 {
			t.Fatalf("session search after delete = %+v, want no results", afterDelete.Results)
		}
		profile, err := svc.Profile(ctx, "user-juan")
		if err != nil {
			t.Fatal(err)
		}
		if len(profile.Card) != 1 {
			t.Fatalf("profile card = %+v, want peer card preserved outside session delete", profile.Card)
		}
	})

	t.Run("recall_trace_diagnostics_replay_and_projection", func(t *testing.T) {
		now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
		engine := newRecallPipelineEngine(staticRecallGenerator{
			candidates: []RecallCandidate{{
				MemoryID:   "101",
				SourceType: "conclusion",
				Content:    "Goncho uses durable RecallTrace before projection.",
				SessionID:  "sess-proof",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.8,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
			}},
			warnings: []RecallWarning{{
				Code:     RecallWarningSemanticUnavailable,
				Stage:    RecallStageGenerate,
				Severity: RecallWarningDegraded,
				Message:  "semantic proof fixture intentionally unavailable",
			}},
		}, recallPipelineOptions{
			pipelineVersion: "proof-pipeline",
			scoringConfig: RecallScoringConfig{
				Version:     "proof-v1",
				Weights:     map[string]float64{"keyword": 1},
				RRFK:        60,
				MMRLambda:   1,
				TokenBudget: 200,
			},
			now: func() time.Time { return now },
		})

		trace, err := engine.Run(context.Background(), RecallQuery{
			WorkspaceID: "default",
			Peer:        "user-juan",
			Query:       "RecallTrace projection",
			SessionKey:  "sess-proof",
			ScopeID:     "team",
			Limit:       1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if trace.TraceID == "" || trace.ScoringConfig.Version != "proof-v1" || len(trace.Selected) != 1 {
			t.Fatalf("trace = %+v, want stable id, scoring config, and selected candidate", trace)
		}
		if !traceHasWarning(trace, RecallWarningSemanticUnavailable) {
			t.Fatalf("trace warnings = %+v, want semantic_unavailable", trace.Warnings)
		}
		raw1, err := trace.StableJSON()
		if err != nil {
			t.Fatal(err)
		}
		raw2, err := trace.StableJSON()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(raw1, raw2) {
			t.Fatalf("StableJSON changed between calls:\n%s\n---\n%s", raw1, raw2)
		}

		diagnostics := BuildRecallDiagnostics(trace)
		if diagnostics.Status != "degraded" || diagnostics.ProjectionInvariant != "no_projection_without_recall_trace" {
			t.Fatalf("diagnostics = %+v, want degraded trace-only invariant", diagnostics)
		}
		replay := BuildRecallReplay(trace)
		if replay.ReplayContract != "deterministic_replay_from_recall_trace" || !recallReplayHasWarning(replay, RecallWarningSemanticUnavailable) {
			t.Fatalf("replay = %+v, want deterministic replay with semantic warning", replay)
		}

		projected := (&RecallProjector{}).ProjectSearch(trace)
		if projected.WorkspaceID != "default" || projected.Peer != "user-juan" || len(projected.Results) != 1 {
			t.Fatalf("projected search = %+v, want trace-derived search result", projected)
		}
	})
}

func recallReplayHasWarning(replay RecallReplay, code string) bool {
	for _, event := range replay.Events {
		if event.WarningCode == code {
			return true
		}
	}
	return false
}
