package status

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

func TestFormatOperatorRunReportTextIncludesDetails(t *testing.T) {
	got := FormatOperatorRunReportText(cron.OperatorRunReport{
		Status:                 "degraded",
		JobID:                  "job-a",
		RunID:                  42,
		Profile:                "dev",
		StartedAtUnix:          1,
		FinishedAtUnix:         2,
		DegradedReason:         "network",
		RecommendedNextCommand: "gormes status",
	})
	for _, want := range []string{"latest run: job-a (run 42) status=degraded", "profile: dev", "degraded: network", "next: gormes status"} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted report missing %q:\n%s", want, got)
		}
	}
}

func TestCollectSystemSnapshotForJSONNormalizesNilSlicesOnError(t *testing.T) {
	got := CollectSystemSnapshotForJSON(nil, Options{})
	if got.Events == nil || got.Presence == nil {
		t.Fatalf("snapshot slices must be non-nil: %+v", got)
	}
}
