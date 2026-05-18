package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
)

func TestGonchoRecallDiagnosticsCommandTextExplainsTrace(t *testing.T) {
	tracePath := filepath.Join("..", "..", "internal", "goncho", "testdata", "recall_trace", "stable_trace.golden.json")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-diagnostics", "--trace", tracePath,
	)
	if err != nil {
		t.Fatalf("goncho recall-diagnostics: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Goncho recall diagnostics",
		"status: ok",
		"trace_id: b3765ad87524b8be6fcdf19dd43e946a8116dd8995e0e998f4a1f08bec69b9ef",
		"pipeline_version: test-pipeline",
		"scoring_config: test-v1",
		"query: auth rate limit",
		"selected candidates",
		"mem-auth",
		"final=0.826218",
		"why: final_score=0.826218; scoring_config=test-v1; diversity_penalty=0.000000",
		"rejected candidates",
		"mem-rate",
		"reason=not_selected",
		"warnings: none",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestGonchoRecallDiagnosticsCommandJSONExposesWarningCodes(t *testing.T) {
	trace := goncho.RecallTrace{
		TraceID:         "trace-warnings",
		PipelineVersion: "test-pipeline",
		Query:           goncho.RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth"},
		ScoringConfig:   goncho.RecallScoringConfig{Version: "warnings-v1", Weights: map[string]float64{"keyword": 1}, RRFK: 60, MMRLambda: 0.7},
		Selected: []goncho.ScoredRecallCandidate{{
			Candidate: goncho.RecallCandidate{MemoryID: "mem-a", SourceType: "turn", Content: "short auth fact"},
			Score:     goncho.RecallScore{FinalScore: 0.7, WhySelected: []string{"final_score=0.700000"}},
		}},
		Warnings: []goncho.RecallWarning{{
			Code:     goncho.RecallWarningTokenBudgetTruncated,
			Stage:    goncho.RecallStageSelect,
			Severity: goncho.RecallWarningDegraded,
			Message:  "token budget truncated selected recall context",
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
		"goncho", "recall-diagnostics", "--trace", tracePath, "--json",
	)
	if err != nil {
		t.Fatalf("goncho recall-diagnostics --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Service string `json:"service"`
		Status  string `json:"status"`
		TraceID string `json:"trace_id"`
		Query   struct {
			Query string `json:"query"`
		} `json:"query"`
		Selected []struct {
			MemoryID   string   `json:"memory_id"`
			FinalScore float64  `json:"final_score"`
			Why        []string `json:"why_selected"`
		} `json:"selected"`
		Warnings []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v\nstdout=%s", err, stdout)
	}
	if got.Service != "goncho" || got.Status != "degraded" || got.TraceID != "trace-warnings" {
		t.Fatalf("header = %+v", got)
	}
	if len(got.Selected) != 1 || got.Selected[0].MemoryID != "mem-a" || got.Selected[0].FinalScore != 0.7 {
		t.Fatalf("selected = %+v", got.Selected)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Code != goncho.RecallWarningTokenBudgetTruncated || got.Warnings[0].Severity != goncho.RecallWarningDegraded {
		t.Fatalf("warnings = %+v", got.Warnings)
	}
}

func TestGonchoRecallDiagnosticsCommandRequiresTraceFile(t *testing.T) {
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-diagnostics",
	)
	if err == nil {
		t.Fatalf("goncho recall-diagnostics without --trace must fail\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error()+stderr, "--trace is required") {
		t.Fatalf("error missing required trace guidance:\nerr=%v\nstderr=%s", err, stderr)
	}
}
