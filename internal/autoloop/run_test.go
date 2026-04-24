package autoloop

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestDryRunSelectsCandidatesWithoutRunningBackend(t *testing.T) {
	progressPath := writeProgressJSON(t, `{
		"phases": {
			"12": {
				"subphases": {
					"12.A": {
						"items": [
							{"item_name": "planned run candidate", "status": "planned"}
						]
					}
				}
			}
		}
	}`)
	runner := &FakeRunner{}

	summary, err := RunOnce(context.Background(), RunOptions{
		Config: Config{
			RepoRoot:     t.TempDir(),
			ProgressJSON: progressPath,
			Backend:      "codexu",
			Mode:         "safe",
			MaxAgents:    1,
		},
		Runner: runner,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	wantSelected := []Candidate{
		{PhaseID: "12", SubphaseID: "12.A", ItemName: "planned run candidate", Status: "planned"},
	}
	if summary.Candidates != 1 {
		t.Fatalf("Candidates = %d, want 1", summary.Candidates)
	}
	if !reflect.DeepEqual(summary.Selected, wantSelected) {
		t.Fatalf("Selected = %#v, want %#v", summary.Selected, wantSelected)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("Commands length = %d, want 0", len(runner.Commands))
	}
}

func TestRunOnceExecutesOncePerSelectedCandidate(t *testing.T) {
	progressPath := writeProgressJSON(t, `{
		"phases": {
			"12": {
				"subphases": {
					"12.A": {
						"items": [
							{"item_name": "active candidate", "status": "in_progress"},
							{"item_name": "planned candidate", "status": "planned"},
							{"item_name": "deferred candidate", "status": "deferred"}
						]
					}
				}
			}
		}
	}`)
	repoRoot := t.TempDir()
	runner := &FakeRunner{
		Results: []Result{{}, {}},
	}

	summary, err := RunOnce(context.Background(), RunOptions{
		Config: Config{
			RepoRoot:     repoRoot,
			ProgressJSON: progressPath,
			Backend:      "opencode",
			Mode:         "safe",
			MaxAgents:    2,
		},
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if summary.Candidates != 3 {
		t.Fatalf("Candidates = %d, want 3", summary.Candidates)
	}
	if got, want := len(summary.Selected), 2; got != want {
		t.Fatalf("Selected length = %d, want %d", got, want)
	}

	wantCommand := Command{Name: "opencode", Args: []string{"run", "--no-interactive"}, Dir: repoRoot}
	wantCommands := []Command{wantCommand, wantCommand}
	if !reflect.DeepEqual(runner.Commands, wantCommands) {
		t.Fatalf("Commands = %#v, want %#v", runner.Commands, wantCommands)
	}
}

func TestRunOnceReturnsBackendRunnerError(t *testing.T) {
	progressPath := writeProgressJSON(t, `{
		"phases": {
			"12": {
				"subphases": {
					"12.A": {
						"items": [
							{"item_name": "planned run candidate", "status": "planned"}
						]
					}
				}
			}
		}
	}`)
	wantErr := errors.New("backend failed")
	runner := &FakeRunner{
		Results: []Result{{Err: wantErr}},
	}

	_, err := RunOnce(context.Background(), RunOptions{
		Config: Config{
			RepoRoot:     t.TempDir(),
			ProgressJSON: progressPath,
			Backend:      "opencode",
			Mode:         "safe",
			MaxAgents:    1,
		},
		Runner: runner,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunOnce() error = %v, want %v", err, wantErr)
	}
}
