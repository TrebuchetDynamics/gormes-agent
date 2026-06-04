package memory

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/service"

	corememory "github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestFormatExtractorStatusPreservesSummary(t *testing.T) {
	out := FormatExtractorStatus(corememory.ExtractorStatus{
		WorkerHealth:    "degraded",
		QueueDepth:      1,
		DeadLetterCount: 2,
		ErrorSummary: []corememory.DeadLetterErrorSummary{
			{Error: "malformed JSON", Count: 1},
		},
		RecentDeadLetters: []corememory.DeadLetterSummary{
			{ID: 3, SessionID: "sess-3", ChatID: "chat", Attempts: 2, Error: "upstream timeout"},
		},
	})
	for _, want := range []string{"Extractor status", "worker_health: degraded", "queue_depth: 1", "dead_letters: 2", "error=\"malformed JSON\" count=1", "session_id=sess-3"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestEmitStatusJSONNormalizesEmptyContainers(t *testing.T) {
	var out bytes.Buffer
	err := EmitStatusJSON(&out, corememory.ExtractorStatus{}, goncho.QueueStatus{}, corememory.Inventory{}, BuildProvenance{Version: "test", GitCommit: "abc"})
	if err != nil {
		t.Fatalf("EmitStatusJSON: %v", err)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Extractor struct {
			ErrorSummary       []any `json:"error_summary"`
			RecentDeadLetters  []any `json:"recent_dead_letters"`
			RecentSkippedSyncs []any `json:"recent_skipped_syncs"`
		} `json:"extractor"`
		GonchoQueue struct {
			WorkUnits map[string]any `json:"work_units"`
		} `json:"goncho_queue"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out.String())
	}
	if got.Build.Version != "test" || got.Extractor.ErrorSummary == nil || got.Extractor.RecentDeadLetters == nil || got.Extractor.RecentSkippedSyncs == nil || got.GonchoQueue.WorkUnits == nil {
		t.Fatalf("json containers not normalized: %+v", got)
	}
}

func TestBuildDefaultsToUnknownProvenance(t *testing.T) {
	got := build(Options{})
	if got.Version != "unknown" || got.GitCommit != "unknown" {
		t.Fatalf("build defaults = %+v, want unknown provenance", got)
	}
}
