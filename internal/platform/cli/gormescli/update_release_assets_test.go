package gormescli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestUpdateReleaseAssetsCommandRunsAssetSkillSyncAfterBinaryUpdate(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	targetVersion := nextPatchVersionForTest(t)
	syncCalled := false
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
			return cli.UpdateReleaseBinaryReport{
				SnapshotID:      "snap-1",
				SnapshotPath:    opts.Plan.SnapshotPath,
				PreviousVersion: currentUpdateVersionForTest(),
				NewVersion:      targetVersion,
				Evidence: []cli.UpdateEvidence{{
					Kind:   cli.UpdateEvidenceReleaseSwapCompleted,
					Detail: "binary swapped",
				}},
			}
		},
		LoadReleaseAssetManifest: func(_ context.Context, plan cli.UpdateReleasePlan) (cli.UpdateReleaseManifest, string, error) {
			if plan.Target.Version != targetVersion {
				t.Fatalf("manifest plan target = %+v, want %s", plan.Target, targetVersion)
			}
			return cli.UpdateReleaseManifest{
				SchemaVersion: 1,
				Skills: []cli.UpdateReleaseSkillManifestEntry{{
					Name: "reviewer",
					Path: "productivity/reviewer/SKILL.md",
				}},
			}, "/tmp/payload", nil
		},
		ReleaseAssetRoot: func() (string, error) {
			return "/tmp/assets", nil
		},
		ReleaseSkillProfiles: func() ([]skills.SkillProfileRoot, error) {
			return []skills.SkillProfileRoot{{Name: "main", Root: "/tmp/profile"}}, nil
		},
		RunReleaseAssetSkillSync: func(_ context.Context, opts cli.UpdateReleaseAssetSkillSyncOptions) cli.UpdateReleaseAssetSkillSyncReport {
			syncCalled = true
			if opts.PayloadRoot != "/tmp/payload" || opts.AssetRoot != "/tmp/assets" || len(opts.SkillProfiles) != 1 {
				t.Fatalf("asset sync opts = %+v, want payload/assets/profile roots", opts)
			}
			if !strings.Contains(opts.SnapshotPath, "assets-skills") {
				t.Fatalf("asset sync snapshot path = %q, want assets-skills child snapshot", opts.SnapshotPath)
			}
			return cli.UpdateReleaseAssetSkillSyncReport{
				SnapshotID:   "assets-skills",
				SnapshotPath: opts.SnapshotPath,
				Evidence: []cli.UpdateEvidence{{
					Kind:   cli.UpdateEvidenceReleaseSkillSyncCompleted,
					Detail: "default updated=1",
				}},
			}
		},
	})

	stdout, stderr, err := executeUpdateCommandForTest(command, "--json")
	if err != nil {
		t.Fatalf("update --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !syncCalled {
		t.Fatal("RunReleaseAssetSkillSync was not called")
	}
	var got struct {
		Evidence []struct {
			Kind string `json:"kind"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", err, stdout)
	}
	found := false
	for _, ev := range got.Evidence {
		if ev.Kind == string(cli.UpdateEvidenceReleaseSkillSyncCompleted) {
			found = true
		}
	}
	if !found {
		t.Fatalf("json evidence = %+v, want skill sync evidence", got.Evidence)
	}
}

func TestUpdateReleaseAssetsCommandRunsWhenBinaryIsCurrent(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	binaryCalled := false
	syncCalled := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		DetectInstallKind: func() cli.UpdateInstallKind {
			return cli.UpdateInstallKindRelease
		},
		RuntimePlatform: func() (string, string) {
			return "linux", "amd64"
		},
		LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
			return cli.UpdateReleaseMetadata{Version: currentUpdateVersionForTest(), Tag: "v" + currentUpdateVersionForTest(), GitCommit: "main"}, nil
		},
		RunReleaseBinaryUpdate: func(context.Context, cli.UpdateReleaseBinaryOptions) cli.UpdateReleaseBinaryReport {
			binaryCalled = true
			return cli.UpdateReleaseBinaryReport{}
		},
		LoadReleaseAssetManifest: func(context.Context, cli.UpdateReleasePlan) (cli.UpdateReleaseManifest, string, error) {
			return cli.UpdateReleaseManifest{
				SchemaVersion: 1,
				Assets: []cli.UpdateReleaseAssetManifestEntry{{
					Path: "web/app.js",
				}},
			}, "/tmp/payload", nil
		},
		ReleaseAssetRoot: func() (string, error) {
			return "/tmp/assets", nil
		},
		RunReleaseAssetSkillSync: func(context.Context, cli.UpdateReleaseAssetSkillSyncOptions) cli.UpdateReleaseAssetSkillSyncReport {
			syncCalled = true
			return cli.UpdateReleaseAssetSkillSyncReport{
				Evidence: []cli.UpdateEvidence{{
					Kind:   cli.UpdateEvidenceReleaseAssetSyncCompleted,
					Detail: "assets updated=1",
				}},
			}
		},
	})

	stdout, stderr, err := executeUpdateCommandForTest(command, "--json")
	if err != nil {
		t.Fatalf("update --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if binaryCalled {
		t.Fatal("current binary must not run binary updater")
	}
	if !syncCalled {
		t.Fatal("asset sync should run when the trusted manifest has asset drift")
	}
	if !strings.Contains(stdout, string(cli.UpdateEvidenceReleaseAssetSyncCompleted)) {
		t.Fatalf("stdout = %s, want asset sync evidence", stdout)
	}
}
