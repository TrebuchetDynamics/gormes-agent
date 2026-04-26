package builderloop

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSelectSpeculativeCandidates(t *testing.T) {
	tests := []struct {
		name               string
		candidates         []Candidate
		completed          map[string]struct{}
		readyWhenSatisfied func(Candidate) bool
		maxSpeculative     int
		wantCount          int
		wantSpeculative    []bool
		wantPending        [][]string
	}{
		{
			name: "selects candidate with blocked_by when ready_when satisfied",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1", BlockedBy: []string{"task-0"}},
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-0", Status: "planned"}, // NOT complete, so it's pending
			},
			completed:          map[string]struct{}{}, // task-0 is NOT completed
			readyWhenSatisfied: func(c Candidate) bool { return true },
			maxSpeculative:     2,
			wantCount:          1,
			wantSpeculative:    []bool{true},
			wantPending:        [][]string{{"task-0"}},
		},
		{
			name: "skips candidate when ready_when not satisfied",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1", BlockedBy: []string{"task-0"}},
			},
			completed:          map[string]struct{}{},
			readyWhenSatisfied: func(c Candidate) bool { return false },
			maxSpeculative:     2,
			wantCount:          0,
		},
		{
			name: "skips candidate with no blocked_by",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1"},
			},
			completed:          map[string]struct{}{},
			readyWhenSatisfied: func(c Candidate) bool { return true },
			maxSpeculative:     2,
			wantCount:          0,
		},
		{
			name: "skips candidate when all blockers complete",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1", BlockedBy: []string{"task-0"}, Status: "complete"},
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-0", Status: "complete"},
			},
			completed:          map[string]struct{}{"task-0": {}, "a/task-0": {}, "1/a/task-0": {}},
			readyWhenSatisfied: func(c Candidate) bool { return true },
			maxSpeculative:     2,
			wantCount:          0,
		},
		{
			name: "respects maxSpeculative limit",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1", BlockedBy: []string{"blocker"}},
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-2", BlockedBy: []string{"blocker"}},
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-3", BlockedBy: []string{"blocker"}},
			},
			completed:          map[string]struct{}{},
			readyWhenSatisfied: func(c Candidate) bool { return true },
			maxSpeculative:     2,
			wantCount:          2,
			wantSpeculative:    []bool{true, true},
			wantPending:        [][]string{{"blocker"}, {"blocker"}},
		},
		{
			name: "zero maxSpeculative returns empty",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1", BlockedBy: []string{"blocker"}},
			},
			completed:          map[string]struct{}{},
			readyWhenSatisfied: func(c Candidate) bool { return true },
			maxSpeculative:     0,
			wantCount:          0,
		},
		{
			name: "handles multiple blockers with partial completion",
			candidates: []Candidate{
				{PhaseID: "1", SubphaseID: "A", ItemName: "task-1", BlockedBy: []string{"blocker-1", "blocker-2"}},
			},
			completed:          map[string]struct{}{"blocker-1": {}},
			readyWhenSatisfied: func(c Candidate) bool { return true },
			maxSpeculative:     2,
			wantCount:          1,
			wantSpeculative:    []bool{true},
			wantPending:        [][]string{{"blocker-2"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectSpeculativeCandidates(tt.candidates, tt.completed, tt.readyWhenSatisfied, tt.maxSpeculative)

			if len(got) != tt.wantCount {
				t.Errorf("selectSpeculativeCandidates() returned %d candidates, want %d", len(got), tt.wantCount)
			}

			for i, c := range got {
				if i < len(tt.wantSpeculative) && c.Speculative != tt.wantSpeculative[i] {
					t.Errorf("candidate %d: Speculative = %v, want %v", i, c.Speculative, tt.wantSpeculative[i])
				}
				if i < len(tt.wantPending) {
					if len(c.BlockedByPending) != len(tt.wantPending[i]) {
						t.Errorf("candidate %d: BlockedByPending length = %d, want %d", i, len(c.BlockedByPending), len(tt.wantPending[i]))
					}
				}
			}
		})
	}
}

func TestEnrichCandidatesWithSpecHash(t *testing.T) {
	hashOf := func(c Candidate) string {
		return "hash-" + c.ItemName
	}

	candidates := []Candidate{
		{PhaseID: "1", SubphaseID: "A", ItemName: "task-1"},
		{PhaseID: "1", SubphaseID: "A", ItemName: "task-2"},
	}

	enriched := enrichCandidatesWithSpecHash(candidates, hashOf)

	if len(enriched) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(enriched))
	}

	if enriched[0].SpecHashAtClaim != "hash-task-1" {
		t.Errorf("candidate 1: SpecHashAtClaim = %q, want %q", enriched[0].SpecHashAtClaim, "hash-task-1")
	}

	if enriched[1].SpecHashAtClaim != "hash-task-2" {
		t.Errorf("candidate 2: SpecHashAtClaim = %q, want %q", enriched[1].SpecHashAtClaim, "hash-task-2")
	}
}

func TestConfigFromEnv_SpeculativeExecution(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantCfg  Config
		wantErr  bool
		errMatch string
	}{
		{
			name: "default values when not set",
			env:  map[string]string{},
			wantCfg: Config{
				SpeculativeExecutionEnabled: false,
				MaxSpeculativeWorkers:       2,
			},
		},
		{
			name: "enables speculative execution",
			env: map[string]string{
				"GORMES_SPECULATIVE_EXECUTION": "true",
			},
			wantCfg: Config{
				SpeculativeExecutionEnabled: true,
			},
		},
		{
			name: "disables speculative execution",
			env: map[string]string{
				"GORMES_SPECULATIVE_EXECUTION": "false",
			},
			wantCfg: Config{
				SpeculativeExecutionEnabled: false,
			},
		},
		{
			name: "sets max speculative workers",
			env: map[string]string{
				"GORMES_MAX_SPECULATIVE_WORKERS": "5",
			},
			wantCfg: Config{
				MaxSpeculativeWorkers: 5,
			},
		},
		{
			name: "rejects negative max speculative workers",
			env: map[string]string{
				"GORMES_MAX_SPECULATIVE_WORKERS": "-1",
			},
			wantErr:  true,
			errMatch: "must be non-negative",
		},
		{
			name: "sets grace period",
			env: map[string]string{
				"GORMES_SPECULATIVE_GRACE_PERIOD": "2h30m",
			},
			wantCfg: Config{
				SpeculativeBlockedByGracePeriod: 150 * time.Minute,
			},
		},
		{
			name: "rejects invalid grace period",
			env: map[string]string{
				"GORMES_SPECULATIVE_GRACE_PERIOD": "invalid",
			},
			wantErr:  true,
			errMatch: "must be a Go duration",
		},
		{
			name: "rejects zero grace period",
			env: map[string]string{
				"GORMES_SPECULATIVE_GRACE_PERIOD": "0",
			},
			wantErr:  true,
			errMatch: "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.env["REPO_ROOT"] = tmpDir

			cfg, err := ConfigFromEnv(tmpDir, MapEnv(tt.env))

			if tt.wantErr {
				if err == nil {
					t.Errorf("ConfigFromEnv() error = nil, want error containing %q", tt.errMatch)
					return
				}
				if !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("ConfigFromEnv() error = %q, want containing %q", err.Error(), tt.errMatch)
				}
				return
			}

			if err != nil {
				t.Errorf("ConfigFromEnv() unexpected error = %v", err)
				return
			}

			if cfg.SpeculativeExecutionEnabled != tt.wantCfg.SpeculativeExecutionEnabled {
				t.Errorf("SpeculativeExecutionEnabled = %v, want %v", cfg.SpeculativeExecutionEnabled, tt.wantCfg.SpeculativeExecutionEnabled)
			}
			if tt.wantCfg.MaxSpeculativeWorkers != 0 && cfg.MaxSpeculativeWorkers != tt.wantCfg.MaxSpeculativeWorkers {
				t.Errorf("MaxSpeculativeWorkers = %d, want %d", cfg.MaxSpeculativeWorkers, tt.wantCfg.MaxSpeculativeWorkers)
			}
			if tt.wantCfg.SpeculativeBlockedByGracePeriod != 0 && cfg.SpeculativeBlockedByGracePeriod != tt.wantCfg.SpeculativeBlockedByGracePeriod {
				t.Errorf("SpeculativeBlockedByGracePeriod = %s, want %s", cfg.SpeculativeBlockedByGracePeriod, tt.wantCfg.SpeculativeBlockedByGracePeriod)
			}
		})
	}
}

func TestCandidate_SelectionReason_Speculative(t *testing.T) {
	tests := []struct {
		name      string
		candidate Candidate
		want      string
	}{
		{
			name:      "non-speculative planned row",
			candidate: Candidate{Status: "planned"},
			want:      "planned row",
		},
		{
			name:      "speculative planned row",
			candidate: Candidate{Status: "planned", Speculative: true},
			want:      "planned row speculative",
		},
		{
			name:      "speculative with penalty",
			candidate: Candidate{Status: "planned", Speculative: true, PenaltyApplied: 5},
			want:      "planned row penalty=5 speculative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.candidate.SelectionReason()
			if !strings.Contains(got, tt.want) {
				t.Errorf("SelectionReason() = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestLedgerEvent_SpeculativeFields(t *testing.T) {
	eventTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	event := LedgerEvent{
		TS:              eventTime,
		RunID:           "run-123",
		Event:           "worker_claimed",
		Worker:          1,
		Task:            "task-1",
		Speculative:     true,
		SpecHashAtClaim: "abc123",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var got LedgerEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got.Speculative != event.Speculative {
		t.Errorf("Speculative = %v, want %v", got.Speculative, event.Speculative)
	}

	if got.SpecHashAtClaim != event.SpecHashAtClaim {
		t.Errorf("SpecHashAtClaim = %q, want %q", got.SpecHashAtClaim, event.SpecHashAtClaim)
	}
}

func TestNormalizeCandidates_WithBlockedBy(t *testing.T) {
	tmpDir := t.TempDir()
	progressPath := filepath.Join(tmpDir, "progress.json")

	progressJSON := `{
		"phases": {
			"1": {
				"subphases": {
					"A": {
						"items": [
							{
								"name": "blocker-task",
								"status": "complete",
								"contract": "Blocker contract"
							},
							{
								"name": "pending-blocker",
								"status": "planned",
								"contract": "Pending blocker contract",
								"contract_status": "fixture_ready"
							},
							{
								"name": "speculative-task",
								"status": "planned",
								"contract": "Speculative contract",
								"contract_status": "fixture_ready",
								"blocked_by": ["pending-blocker"]
							}
						]
					}
				}
			}
		}
	}`

	if err := os.WriteFile(progressPath, []byte(progressJSON), 0644); err != nil {
		t.Fatalf("failed to write progress.json: %v", err)
	}

	candidates, err := NormalizeCandidates(progressPath, CandidateOptions{
		ActiveFirst: true,
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates failed: %v", err)
	}

	var foundBlocker, foundPending, foundSpeculative bool
	for _, c := range candidates {
		switch c.ItemName {
		case "blocker-task":
			foundBlocker = true
		case "pending-blocker":
			foundPending = true
		case "speculative-task":
			foundSpeculative = true
		}
	}

	if foundSpeculative {
		t.Error("speculative-task should be filtered out due to blocked_by")
	}

	if !foundPending {
		t.Error("pending-blocker should be included")
	}

	if foundBlocker {
		t.Error("blocker-task should be filtered out (status=complete)")
	}
}

func TestSpeculativeExecution_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	repoRoot := tmpDir

	execGit := func(args ...string) error {
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		return cmd.Run()
	}

	if err := execGit("init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := execGit("config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := execGit("config", "user.name", "Test"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}

	progressDir := filepath.Join(repoRoot, "docs", "content", "building-gormes", "architecture_plan")
	if err := os.MkdirAll(progressDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	progressJSON := `{
		"phases": {
			"1": {
				"subphases": {
					"A": {
						"items": [
							{
								"name": "blocker",
								"status": "planned",
								"contract": "Blocker contract",
								"contract_status": "fixture_ready"
							},
							{
								"name": "dependent",
								"status": "planned",
								"contract": "Dependent contract",
								"contract_status": "fixture_ready",
								"blocked_by": ["blocker"]
							}
						]
					}
				}
			}
		}
	}`

	progressPath := filepath.Join(progressDir, "progress.json")
	if err := os.WriteFile(progressPath, []byte(progressJSON), 0644); err != nil {
		t.Fatalf("failed to write progress.json: %v", err)
	}

	if err := execGit("add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := execGit("commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// First verify blocker is included (it has fixture_ready contract status)
	candidates, err := NormalizeCandidates(progressPath, CandidateOptions{
		ActiveFirst: true,
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates failed: %v", err)
	}

	// Only "blocker" should be returned since "dependent" is blocked
	if len(candidates) != 1 {
		t.Errorf("expected 1 candidate (blocker only), got %d", len(candidates))
	}

	// Now test speculative selection with IncludeBlocked to get both
	allCandidates, err := NormalizeCandidates(progressPath, CandidateOptions{
		ActiveFirst:    true,
		IncludeBlocked: true,
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates with IncludeBlocked failed: %v", err)
	}

	if len(allCandidates) != 2 {
		t.Errorf("expected 2 candidates with IncludeBlocked, got %d", len(allCandidates))
	}

	completed := make(map[string]struct{})

	speculative := selectSpeculativeCandidates(allCandidates, completed, func(c Candidate) bool {
		return true
	}, 2)

	if len(speculative) != 1 {
		t.Errorf("expected 1 speculative candidate, got %d", len(speculative))
	}

	if len(speculative) > 0 {
		if speculative[0].ItemName != "dependent" {
			t.Errorf("expected speculative candidate 'dependent', got %q", speculative[0].ItemName)
		}
		if !speculative[0].Speculative {
			t.Error("expected Speculative flag to be true")
		}
		if len(speculative[0].BlockedByPending) != 1 || speculative[0].BlockedByPending[0] != "blocker" {
			t.Errorf("expected BlockedByPending=['blocker'], got %v", speculative[0].BlockedByPending)
		}
	}
}

func TestRunOnceDryRunSelectsBlockedRowsWhenSpeculativeExecutionEnabled(t *testing.T) {
	progressPath := writeProgressJSON(t, `{
		"phases": {
			"1": {
				"subphases": {
					"1.A": {
						"items": [
							{
								"name": "base-ready",
								"status": "planned",
								"contract": "Base contract",
								"contract_status": "draft"
							},
							{
								"name": "blocked-ready",
								"status": "planned",
								"contract": "Blocked contract",
								"contract_status": "draft",
								"blocked_by": ["base-ready"],
								"ready_when": ["base-ready is claimed this run"]
							}
						]
					}
				}
			}
		}
	}`)

	summary, err := RunOnce(context.Background(), RunOptions{
		Config: Config{
			RepoRoot:                        t.TempDir(),
			ProgressJSON:                    progressPath,
			Backend:                         "codexu",
			Mode:                            "safe",
			MaxAgents:                       2,
			SpeculativeExecutionEnabled:     true,
			MaxSpeculativeWorkers:           1,
			MergeOpenPullRequests:           false,
			PostPromotionVerifyCommands:     nil,
			PostPromotionRepairEnabled:      false,
			PostPromotionRepairAttempts:     0,
			SpeculativeBlockedByGracePeriod: time.Hour,
		},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if got, want := len(summary.Selected), 2; got != want {
		t.Fatalf("selected length = %d, want %d: %#v", got, want, summary.Selected)
	}
	if got, want := summary.Candidates, 2; got != want {
		t.Fatalf("candidate count = %d, want %d", got, want)
	}
	if got, want := summary.Selected[1].ItemName, "blocked-ready"; got != want {
		t.Fatalf("second selected item = %q, want %q", got, want)
	}
	if !summary.Selected[1].Speculative {
		t.Fatalf("blocked-ready Speculative = false, want true")
	}
	if got, want := summary.Selected[1].BlockedByPending, []string{"base-ready"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("blocked-ready BlockedByPending = %#v, want %#v", got, want)
	}
	if summary.Selected[1].SpecHashAtClaim == "" {
		t.Fatalf("blocked-ready SpecHashAtClaim is empty")
	}
}
