package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestUpdateReleaseBinaryCommandUsesReleaseUpdaterForReleaseInstall(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	targetVersion := nextPatchVersionForTest(t)
	updateCalled := false
	lifecycleCalled := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		DetectInstallKind: func() cli.UpdateInstallKind {
			return cli.UpdateInstallKindRelease
		},
		RuntimePlatform: func() (string, string) {
			return "linux", "amd64"
		},
		LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
			return nextPatchReleaseMetadataForTest(t, "abc1234"), nil
		},
		RunReleaseBinaryUpdate: func(_ context.Context, opts cli.UpdateReleaseBinaryOptions) cli.UpdateReleaseBinaryReport {
			updateCalled = true
			if opts.Plan.Source != cli.UpdateSourceGitHubRelease || opts.Plan.ArtifactName != nextPatchReleaseArtifactForTest(t, "linux", "amd64") {
				t.Fatalf("release plan = %+v, want GitHub release linux-amd64 artifact", opts.Plan)
			}
			if opts.ManagedBinPath == "" || opts.PublishedBinPath == "" {
				t.Fatalf("binary paths not wired: managed=%q published=%q", opts.ManagedBinPath, opts.PublishedBinPath)
			}
			return cli.UpdateReleaseBinaryReport{
				SnapshotID:       "snap-1",
				SnapshotPath:     "/tmp/snap-1",
				PreviousVersion:  Version,
				NewVersion:       targetVersion,
				ManagedBinPath:   opts.ManagedBinPath,
				PublishedBinPath: opts.PublishedBinPath,
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceReleaseSwapCompleted, Detail: "published"},
				},
			}
		},
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			lifecycleCalled = true
			return cli.UpdateReport{}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--json")
	if err != nil {
		t.Fatalf("update --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !updateCalled || lifecycleCalled {
		t.Fatalf("updateCalled=%t lifecycleCalled=%t, want release updater only", updateCalled, lifecycleCalled)
	}
	var got struct {
		Action     string `json:"action"`
		SnapshotID string `json:"snapshot_id"`
		NewVersion string `json:"new_version"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", err, stdout)
	}
	if got.Action != "update_release_binary" || got.SnapshotID != "snap-1" || got.NewVersion != targetVersion {
		t.Fatalf("json = %+v, want release binary report", got)
	}
}

func TestUpdateReleaseRollbackCommandUsesSnapshotID(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	rollbackCalled := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		RunReleaseRollback: func(_ context.Context, opts cli.UpdateReleaseRollbackOptions) cli.UpdateReleaseBinaryReport {
			rollbackCalled = true
			if opts.SnapshotID != "snap-1" || !strings.Contains(opts.SnapshotRoot, "snapshots") {
				t.Fatalf("rollback opts = %+v, want snapshot id under install snapshots", opts)
			}
			return cli.UpdateReleaseBinaryReport{
				SnapshotID:   opts.SnapshotID,
				SnapshotPath: opts.SnapshotPath,
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceReleaseRollbackCompleted, Detail: "restored"},
				},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--rollback", "snap-1", "--json")
	if err != nil {
		t.Fatalf("update --rollback --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !rollbackCalled {
		t.Fatal("RunReleaseRollback was not called")
	}
	var got struct {
		Action     string `json:"action"`
		SnapshotID string `json:"snapshot_id"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", err, stdout)
	}
	if got.Action != "update_rollback" || got.SnapshotID != "snap-1" {
		t.Fatalf("json = %+v, want rollback report", got)
	}
}
