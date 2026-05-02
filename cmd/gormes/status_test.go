package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootStatusCommandShowsProgressBlockers(t *testing.T) {
	path := writeRootStatusProgressFixture(t, `{
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
                "type": "infra",
                "status": "blocker_active",
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

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "status", "--progress", path)
	if err != nil {
		t.Fatalf("gormes status failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Status",
		"blockers: 1 active",
		"5/5.N blocked row type=infra owner=operator status=blocker_active",
		"evidence: sessions.db locked",
		"workaround/pivot: run next P0 row",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("gormes status output missing %q:\n%s", want, stdout)
		}
	}
}

func writeRootStatusProgressFixture(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}
