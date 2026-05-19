package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho"
)

func TestGonchoRecallReplayCommandTextShowsTimeline(t *testing.T) {
	tracePath := filepath.Join("..", "..", "internal", "goncho", "testdata", "recall_trace", "stable_trace.golden.json")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-replay", "--trace", tracePath,
	)
	if err != nil {
		t.Fatalf("goncho recall-replay: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Goncho recall replay",
		"trace_id: b3765ad87524b8be6fcdf19dd43e946a8116dd8995e0e998f4a1f08bec69b9ef",
		"pipeline_version: test-pipeline",
		"scoring_config: test-v1",
		"query: auth rate limit",
		"candidate_scored memory_id=mem-auth",
		"selected memory_id=mem-auth final=0.826218",
		"rejected memory_id=mem-rate reason=not_selected final=0.428847",
		"projection_ready trace_only=true selected=2 rejected=1 warnings=0",
		"warnings: none",
		"projection_invariant: no_projection_without_recall_trace",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestGonchoRecallReplayCommandJSONExposesReplayEvents(t *testing.T) {
	trace := goncho.RecallTrace{
		TraceID:         "trace-replay-json",
		PipelineVersion: "test-pipeline",
		Query:           goncho.RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth"},
		ScoringConfig:   goncho.RecallScoringConfig{Version: "json-v1", Weights: map[string]float64{"keyword": 1}, RRFK: 60, MMRLambda: 0.7},
		Candidates: []goncho.ScoredRecallCandidate{{
			Candidate: goncho.RecallCandidate{MemoryID: "mem-a", SourceType: "turn", Content: "short auth fact"},
			Score:     goncho.RecallScore{FinalScore: 0.7, WhySelected: []string{"final_score=0.700000"}},
		}},
		Selected: []goncho.ScoredRecallCandidate{{
			Candidate: goncho.RecallCandidate{MemoryID: "mem-a", SourceType: "turn", Content: "short auth fact"},
			Score:     goncho.RecallScore{FinalScore: 0.7, WhySelected: []string{"final_score=0.700000"}},
		}},
		Warnings: []goncho.RecallWarning{{
			Code:     goncho.RecallWarningSemanticUnavailable,
			Stage:    goncho.RecallStageGenerate,
			Severity: goncho.RecallWarningDegraded,
			Message:  "semantic generator unavailable",
		}},
	}
	raw, err := json.Marshal(trace)
	if err != nil {
		t.Fatal(err)
	}
	tracePath := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tracePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-replay", "--trace", tracePath, "--json",
	)
	if err != nil {
		t.Fatalf("goncho recall-replay --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Service              string `json:"service"`
		TraceID              string `json:"trace_id"`
		PipelineVersion      string `json:"pipeline_version"`
		ScoringConfigVersion string `json:"scoring_config_version"`
		EventCount           int    `json:"event_count"`
		ProjectionInvariant  string `json:"projection_invariant"`
		ReplayContract       string `json:"replay_contract"`
		Events               []struct {
			Index       int      `json:"index"`
			Stage       string   `json:"stage"`
			Kind        string   `json:"kind"`
			MemoryID    string   `json:"memory_id"`
			WarningCode string   `json:"warning_code"`
			Severity    string   `json:"severity"`
			Details     []string `json:"details"`
		} `json:"events"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v\nstdout=%s", err, stdout)
	}
	if got.Service != "goncho" || got.TraceID != "trace-replay-json" || got.PipelineVersion != "test-pipeline" || got.ScoringConfigVersion != "json-v1" {
		t.Fatalf("header = %+v", got)
	}
	if got.ProjectionInvariant != "no_projection_without_recall_trace" || got.ReplayContract != "deterministic_replay_from_recall_trace" {
		t.Fatalf("contracts = projection %q replay %q", got.ProjectionInvariant, got.ReplayContract)
	}
	if got.EventCount != len(got.Events) || got.EventCount != 5 {
		t.Fatalf("event count = %d len(events)=%d events=%+v", got.EventCount, len(got.Events), got.Events)
	}
	if got.Events[0].Kind != "recall_query" || got.Events[1].Kind != "candidate_scored" || got.Events[2].WarningCode != goncho.RecallWarningSemanticUnavailable || got.Events[3].MemoryID != "mem-a" || got.Events[4].Kind != "projection_ready" {
		t.Fatalf("events = %+v", got.Events)
	}
}

func TestGonchoRecallReplayCommandRequiresTraceFile(t *testing.T) {
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-replay",
	)
	if err == nil {
		t.Fatalf("goncho recall-replay without --trace must fail\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error()+stderr, "--trace is required") {
		t.Fatalf("error missing required trace guidance:\nerr=%v\nstderr=%s", err, stderr)
	}
}
