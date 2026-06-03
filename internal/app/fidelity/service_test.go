package fidelity

import (
	"strings"
	"testing"

	progressfidelity "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
)

func TestFormatFidelityHermesReportPreservesTextContract(t *testing.T) {
	got := formatFidelityHermesReport(fidelityHermesReportJSON{
		OK: true,
		Report: progressfidelity.Report{
			HermesSHA: "abc123",
			Summary: progressfidelity.Summary{Total: 1, ByStatus: map[string]int{
				string(progressfidelity.StatusCovered): 1,
			}},
			Surfaces: []progressfidelity.SurfaceReport{{ID: "goncho_memory", Status: progressfidelity.StatusCovered}},
		},
	})
	for _, want := range []string{"Hermes fidelity report", "status: true", "hermes_sha: abc123", "surfaces: total=1 covered=1", "- goncho_memory status=covered"} {
		if !strings.Contains(got, want) {
			t.Fatalf("report missing %q:\n%s", want, got)
		}
	}
}
