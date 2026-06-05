package statusreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderStatusReportShowsProgressBlockers(t *testing.T) {
	path := writeStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {
    "5": {
      "name": "Phase 5",
      "deliverable": "tools",
      "subphases": {
        "5.N": {
          "name": "operator tools",
          "items": [
            {
              "name": "blocked row",
              "status": "planned",
              "blocker": {
                "title": "blocked row",
                "type": "infra",
                "status": "blocker_active",
                "recorded_at": "2026-05-01T12:00:00-06:00",
                "blocker": "gateway lock",
                "evidence": "sessions.db locked",
                "unblocks_when": "lock exits",
                "owner": "operator",
                "pivot": "run next P0 row",
                "next_check": "2026-05-01T12:30:00-06:00"
              }
            }
          ]
        }
      }
    }
  }
}`)

	got, err := RenderStatusReport(StatusReportOptions{ProgressPath: path})
	if err != nil {
		t.Fatalf("RenderStatusReport: %v", err)
	}
	for _, want := range []string{
		"Gormes Status",
		"blockers: 1 active",
		"5/5.N blocked row type=infra owner=operator status=blocker_active",
		"evidence: sessions.db locked",
		"workaround/pivot: run next P0 row",
		"next check: 2026-05-01T12:30:00-06:00",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("status report missing %q:\n%s", want, got)
		}
	}
}

func TestRenderStatusReportDegradesWhenProgressMissing(t *testing.T) {
	got, err := RenderStatusReport(StatusReportOptions{ProgressPath: "/does/not/exist/progress.json"})
	if err != nil {
		t.Fatalf("RenderStatusReport should render degraded status, got error: %v", err)
	}
	if !strings.Contains(got, "blockers: unavailable") || !strings.Contains(got, "progress_unavailable") {
		t.Fatalf("missing degraded blocker status:\n%s", got)
	}
}

func writeStatusProgressFixture(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}
