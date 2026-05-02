package progress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgressBlockerMetadataRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	body := `{
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
              "contract": "do work",
              "contract_status": "draft",
              "slice_size": "small",
              "execution_owner": "tools",
              "trust_class": ["operator"],
              "degraded_mode": "visible",
              "fixture": "internal/tools/blocker_test.go",
              "source_refs": ["AGENTS.md"],
              "ready_when": ["operator resolves blocker"],
              "acceptance": ["status shows blocker"],
              "write_scope": ["internal/tools/blocker.go"],
              "test_commands": ["go test ./internal/tools -run TestBlocker -count=1"],
              "done_signal": ["blocker visible"],
              "wired": true,
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
}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	prog, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	row := prog.Phases["5"].Subphases["5.N"].Items[0]
	if row.Blocker == nil {
		t.Fatal("Blocker metadata was not parsed")
	}
	if row.Blocker.Type != "infra" || row.Blocker.Owner != "operator" || row.Blocker.Pivot != "run next P0 row" {
		t.Fatalf("Blocker metadata = %+v", row.Blocker)
	}

	if err := SaveProgress(path, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	roundTripped, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, want := range []string{`"blocker"`, `"type": "infra"`, `"pivot": "run next P0 row"`} {
		if !strings.Contains(string(roundTripped), want) {
			t.Fatalf("round-tripped progress missing %q:\n%s", want, roundTripped)
		}
	}
	if !strings.Contains(string(roundTripped), `"wired": true`) {
		t.Fatalf("round-tripped progress missing wired evidence:\n%s", roundTripped)
	}
}

func TestValidateRejectsBlockerMetadataWithoutPivot(t *testing.T) {
	p := &Progress{
		Meta: Meta{Version: "2.0"},
		Phases: map[string]Phase{"5": {Subphases: map[string]Subphase{
			"5.N": {Items: []Item{{
				Name:           "blocked",
				Status:         StatusPlanned,
				Contract:       "blocked",
				ContractStatus: ContractStatusDraft,
				SliceSize:      SliceSizeSmall,
				ExecutionOwner: ExecutionOwnerTools,
				TrustClass:     []string{"operator"},
				DegradedMode:   "visible",
				Fixture:        "internal/tools/blocker_test.go",
				SourceRefs:     []string{"AGENTS.md"},
				ReadyWhen:      []string{"ready"},
				Acceptance:     []string{"status"},
				WriteScope:     []string{"internal/tools/blocker.go"},
				TestCommands:   []string{"go test ./internal/tools -run TestBlocker -count=1"},
				DoneSignal:     []string{"done"},
				Blocker: &BlockerMetadata{
					Type:          "infra",
					Blocker:       "lock",
					Evidence:      "locked",
					UnblocksWhen:  "unlocked",
					Owner:         "operator",
					NextCheck:     "2026-05-01T12:30:00-06:00",
					MissingFields: []string{"pivot"},
				},
			}}},
		}}},
	}

	err := Validate(p)
	if err == nil || !strings.Contains(err.Error(), "blocker metadata missing pivot") {
		t.Fatalf("Validate() = %v, want blocker metadata missing pivot", err)
	}
}
