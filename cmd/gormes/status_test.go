package main

import (
	"encoding/json"
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

// TestRootStatusCommand_JSONEmitsStructuredBlockers proves
// `gormes status --json` returns a parseable {"blockers": [...]}
// document with phase, subphase, row, and the BlockerRecord fields.
// CI/cron consumers monitoring active blockers can ingest this without
// scraping the human-readable rows.
func TestRootStatusCommand_JSONEmitsStructuredBlockers(t *testing.T) {
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
                "next_check": "2026-05-08T12:30:00-06:00"
              }
            }
          ]
        }
      }
    }
  }
}`)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "status", "--progress", path, "--json")
	if err != nil {
		t.Fatalf("gormes status --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}

	var got struct {
		Blockers []struct {
			Phase    string `json:"phase"`
			Subphase string `json:"subphase"`
			Row      string `json:"row"`
			Title    string `json:"title"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Blocker  string `json:"blocker"`
			Evidence string `json:"evidence"`
			Owner    string `json:"owner"`
		} `json:"blockers"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if len(got.Blockers) != 1 {
		t.Fatalf("got %d blockers, want 1", len(got.Blockers))
	}
	b := got.Blockers[0]
	if b.Phase != "5" || b.Subphase != "5.N" || b.Row != "blocked row" {
		t.Fatalf("unexpected coordinates: phase=%q subphase=%q row=%q", b.Phase, b.Subphase, b.Row)
	}
	if b.Type != "infra" || b.Status != "blocker_active" || b.Owner != "operator" {
		t.Fatalf("unexpected record: type=%q status=%q owner=%q", b.Type, b.Status, b.Owner)
	}
	// JSON mode must not also emit the human header.
	if strings.Contains(stdout, "Gormes Status") {
		t.Fatalf("--json must not emit the human header; got:\n%s", stdout)
	}
}

// TestRootStatusCommand_JSONNoBlockersEmitsEmptyArray proves the JSON
// surface stays parseable when no blockers exist — consumers see
// `{"blockers": []}`, not a free-form "blockers: none" message.
func TestRootStatusCommand_JSONNoBlockersEmitsEmptyArray(t *testing.T) {
	path := writeRootStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(cmd, "status", "--progress", path, "--json")
	if err != nil {
		t.Fatalf("gormes status --json: %v", err)
	}
	var got struct {
		Blockers []any `json:"blockers"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Blockers == nil {
		t.Fatalf("blockers must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Blockers) != 0 {
		t.Fatalf("got %d blockers, want 0", len(got.Blockers))
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
