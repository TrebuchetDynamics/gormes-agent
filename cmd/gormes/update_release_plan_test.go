package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestUpdateCommandReleaseDryRunPrintsPlanWithoutLifecycleMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	installHome := t.TempDir()
	t.Setenv("GORMES_INSTALL_HOME", installHome)
	targetVersion := nextPatchVersionForTest(t)

	lifecycleCalled := false
	checkoutDirCalled := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) {
			checkoutDirCalled = true
			return "/repo/gormes", nil
		},
		DetectInstallKind: func() cli.UpdateInstallKind {
			return cli.UpdateInstallKindRelease
		},
		RuntimePlatform: func() (string, string) {
			return "linux", "amd64"
		},
		LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
			return nextPatchReleaseMetadataForTest(t, "newsha"), nil
		},
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			lifecycleCalled = true
			return cli.UpdateReport{}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--dry-run", "--json")
	if err != nil {
		t.Fatalf("update --dry-run --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if lifecycleCalled || checkoutDirCalled {
		t.Fatalf("dry-run touched source lifecycle: lifecycle=%t checkoutDir=%t", lifecycleCalled, checkoutDirCalled)
	}
	if _, err := os.Stat(filepath.Join(installHome, "lifecycle", "update.log")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run must not write update.log; stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(installHome, "lifecycle", "install.log.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run must not append install.log.jsonl; stat err=%v", err)
	}

	var got struct {
		Action string `json:"action"`
		Failed bool   `json:"failed"`
		Plan   struct {
			Source          string   `json:"source"`
			InstallKind     string   `json:"install_kind"`
			Channel         string   `json:"channel"`
			ArtifactName    string   `json:"artifact_name"`
			SnapshotPath    string   `json:"snapshot_path"`
			UpdateAvailable bool     `json:"update_available"`
			Components      []string `json:"components"`
			Current         struct {
				Version string `json:"version"`
			} `json:"current"`
			Target struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
			} `json:"target"`
		} `json:"plan"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", err, stdout)
	}
	if got.Action != "update_dry_run" || got.Failed {
		t.Fatalf("header = action %q failed %t, want update_dry_run success", got.Action, got.Failed)
	}
	if got.Plan.Source != "github_release" || got.Plan.InstallKind != "release" || got.Plan.Channel != "stable" {
		t.Fatalf("plan route = source %q kind %q channel %q, want release stable GitHub plan", got.Plan.Source, got.Plan.InstallKind, got.Plan.Channel)
	}
	if got.Plan.Current.Version != Version || got.Plan.Target.Version != targetVersion || got.Plan.Target.GitCommit != "newsha" {
		t.Fatalf("plan identity = current %q target %+v, want running build -> release metadata", got.Plan.Current.Version, got.Plan.Target)
	}
	if got.Plan.ArtifactName != nextPatchReleaseArtifactForTest(t, "linux", "amd64") {
		t.Fatalf("artifact = %q, want linux-amd64 release archive", got.Plan.ArtifactName)
	}
	if !strings.Contains(got.Plan.SnapshotPath, "snapshots") {
		t.Fatalf("snapshot path = %q, want pre-update snapshots path", got.Plan.SnapshotPath)
	}
	if len(got.Plan.Components) == 0 {
		t.Fatalf("components empty; got %+v", got.Plan)
	}
}

func TestUpdateCommandCheckReleaseExitCodesAndJSON(t *testing.T) {
	for _, tc := range []struct {
		name       string
		target     cli.UpdateReleaseMetadata
		wantCode   int
		wantUpdate bool
	}{
		{
			name:       "current release exits zero",
			target:     cli.UpdateReleaseMetadata{Version: Version, Tag: "v" + Version, GitCommit: "main"},
			wantCode:   0,
			wantUpdate: false,
		},
		{
			name:       "available release exits ten",
			target:     cli.UpdateReleaseMetadata{Version: "999.0.0", Tag: "v999.0.0", GitCommit: "future"},
			wantCode:   10,
			wantUpdate: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupOneshotFlagTestEnv(t)
			t.Setenv("GORMES_INSTALL_HOME", t.TempDir())

			command := newUpdateCommandWithSeams(updateCommandSeams{
				DetectInstallKind: func() cli.UpdateInstallKind {
					return cli.UpdateInstallKindRelease
				},
				RuntimePlatform: func() (string, string) {
					return "linux", "amd64"
				},
				LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
					return tc.target, nil
				},
				RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
					t.Fatal("release --check must not run source lifecycle")
					return cli.UpdateReport{}
				},
			})

			stdout, stderr, err := executeRootCommandForTest(command, "--check", "--json")
			if code := exitCodeFromError(err); code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d err=%v stderr=%s stdout=%s", code, tc.wantCode, err, stderr, stdout)
			}
			var got struct {
				Action string `json:"action"`
				Failed bool   `json:"failed"`
				Plan   struct {
					UpdateAvailable bool `json:"update_available"`
				} `json:"plan"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
				t.Fatalf("stdout must be valid JSON: %v\n%s", jsonErr, stdout)
			}
			if got.Action != "update_check" || got.Failed {
				t.Fatalf("header = action %q failed %t, want update_check success", got.Action, got.Failed)
			}
			if got.Plan.UpdateAvailable != tc.wantUpdate {
				t.Fatalf("update_available = %t, want %t", got.Plan.UpdateAvailable, tc.wantUpdate)
			}
		})
	}
}

func TestUpdateCommandCheckReleaseMetadataFailureReturnsNonZero(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	command := newUpdateCommandWithSeams(updateCommandSeams{
		DetectInstallKind: func() cli.UpdateInstallKind {
			return cli.UpdateInstallKindRelease
		},
		RuntimePlatform: func() (string, string) {
			return "linux", "amd64"
		},
		LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
			return cli.UpdateReleaseMetadata{}, errors.New("github API unavailable")
		},
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			t.Fatal("release --check must not run source lifecycle when metadata lookup fails")
			return cli.UpdateReport{}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--check", "--json")
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1 err=%v stderr=%s stdout=%s", code, err, stderr, stdout)
	}
	var got struct {
		Failed bool `json:"failed"`
		Plan   struct {
			Blockers []struct {
				Kind string `json:"kind"`
			} `json:"blockers"`
		} `json:"plan"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", jsonErr, stdout)
	}
	if !got.Failed {
		t.Fatalf("metadata failure must mark report failed; got %+v", got)
	}
	found := false
	for _, blocker := range got.Plan.Blockers {
		if blocker.Kind == "missing_release_metadata" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("blockers = %+v, want missing_release_metadata", got.Plan.Blockers)
	}
}
