package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestUpdateReleaseAssetSyncRejectsInvalidManifestBeforeMutation(t *testing.T) {
	root := t.TempDir()
	payloadRoot := filepath.Join(root, "payload")
	assetRoot := filepath.Join(root, "assets")
	writeReleasePayloadFile(t, payloadRoot, "assets/app.js", "console.log('new')")
	existingAsset := writeReleasePayloadFile(t, assetRoot, "app.js", "console.log('old')")
	snapshotPath := filepath.Join(root, "snapshots", "asset-sync")

	report := RunUpdateReleaseAssetSkillSync(context.Background(), UpdateReleaseAssetSkillSyncOptions{
		Plan:         testReleaseBinaryPlan(snapshotPath),
		PayloadRoot:  payloadRoot,
		AssetRoot:    assetRoot,
		SnapshotPath: snapshotPath,
		Manifest: UpdateReleaseManifest{
			SchemaVersion: 1,
			Assets: []UpdateReleaseAssetManifestEntry{{
				Path:        "../escape.js",
				PayloadPath: "assets/app.js",
				SHA256:      releaseTestSHA256("console.log('new')"),
			}},
		},
	})

	if !report.Failed {
		t.Fatalf("invalid manifest should fail before mutation: %+v", report)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseManifestFailed)
	assertFileBody(t, existingAsset, "console.log('old')")
	if _, err := os.Stat(snapshotPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid manifest must not create snapshot; stat err=%v", err)
	}
}

func TestUpdateReleaseAssetSyncAppliesVerifiedAssetsAndRollbackRestores(t *testing.T) {
	root := t.TempDir()
	payloadRoot := filepath.Join(root, "payload")
	assetRoot := filepath.Join(root, "assets")
	writeReleasePayloadFile(t, payloadRoot, "web/app.js", "console.log('new')")
	target := writeReleasePayloadFile(t, assetRoot, "web/app.js", "console.log('old')")
	snapshotPath := filepath.Join(root, "snapshots", "asset-sync")

	report := RunUpdateReleaseAssetSkillSync(context.Background(), UpdateReleaseAssetSkillSyncOptions{
		Plan:         testReleaseBinaryPlan(snapshotPath),
		PayloadRoot:  payloadRoot,
		AssetRoot:    assetRoot,
		SnapshotPath: snapshotPath,
		Manifest: UpdateReleaseManifest{
			SchemaVersion: 1,
			Assets: []UpdateReleaseAssetManifestEntry{{
				Path:        "web/app.js",
				PayloadPath: "web/app.js",
				SHA256:      releaseTestSHA256("console.log('new')"),
			}},
		},
	})
	if report.Failed {
		t.Fatalf("RunUpdateReleaseAssetSkillSync failed: %+v", report)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseAssetSyncCompleted)
	assertFileBody(t, target, "console.log('new')")
	if report.SnapshotID != "asset-sync" || report.SnapshotPath != snapshotPath {
		t.Fatalf("snapshot = %q/%q, want asset-sync at planned path", report.SnapshotID, report.SnapshotPath)
	}

	rollback := RunUpdateReleaseAssetSkillRollback(context.Background(), UpdateReleaseAssetSkillRollbackOptions{
		SnapshotPath: snapshotPath,
	})
	if rollback.Failed {
		t.Fatalf("rollback failed: %+v", rollback)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: rollback.Evidence}, UpdateEvidenceReleaseAssetRollbackCompleted)
	assertFileBody(t, target, "console.log('old')")
}

func TestUpdateReleaseSkillSyncSnapshotsProfilesAndRollbackPreservesPostUpdateEdits(t *testing.T) {
	root := t.TempDir()
	payloadRoot := filepath.Join(root, "payload")
	defaultRoot := filepath.Join(root, "profiles", "default")
	workRoot := filepath.Join(root, "profiles", "work")
	oldBody := releaseSkillDoc("reviewer", "old review")
	newBody := releaseSkillDoc("reviewer", "new review")
	writeReleasePayloadFile(t, payloadRoot, "skills/reviewer/SKILL.md", newBody)
	defaultTarget := writeReleasePayloadFile(t, defaultRoot, "skills/active/productivity/reviewer/SKILL.md", oldBody)
	workTarget := writeReleasePayloadFile(t, workRoot, "skills/active/productivity/reviewer/SKILL.md", "operator edited reviewer")
	snapshotPath := filepath.Join(root, "snapshots", "skill-sync")

	report := RunUpdateReleaseAssetSkillSync(context.Background(), UpdateReleaseAssetSkillSyncOptions{
		Plan:         testReleaseBinaryPlan(snapshotPath),
		PayloadRoot:  payloadRoot,
		SnapshotPath: snapshotPath,
		SkillProfiles: []skills.SkillProfileRoot{
			{Name: "work", Root: workRoot},
			{Name: "default", Root: defaultRoot},
		},
		Manifest: UpdateReleaseManifest{
			SchemaVersion: 1,
			Skills: []UpdateReleaseSkillManifestEntry{{
				Name:           "reviewer",
				Path:           "productivity/reviewer/SKILL.md",
				PayloadPath:    "skills/reviewer/SKILL.md",
				SHA256:         releaseTestSHA256(newBody),
				PreviousSHA256: releaseTestSHA256(oldBody),
			}},
		},
	})
	if report.Failed {
		t.Fatalf("RunUpdateReleaseAssetSkillSync failed: %+v", report)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseSkillSyncCompleted)
	assertFileBody(t, defaultTarget, newBody)
	assertFileBody(t, workTarget, "operator edited reviewer")
	conflictPath := filepath.Join(workRoot, "skills", ".bundled-conflicts", "reviewer", releaseTestSHA256(newBody)[:12], "productivity", "reviewer", "SKILL.md")
	assertFileBody(t, conflictPath, newBody)
	if report.SkillSummaries[0].Profile != "default" || report.SkillSummaries[0].Updated != 1 {
		t.Fatalf("default skill summary = %+v, want one updated", report.SkillSummaries[0])
	}
	if report.SkillSummaries[1].Profile != "work" || report.SkillSummaries[1].ConflictCopies != 1 {
		t.Fatalf("work skill summary = %+v, want one conflict copy", report.SkillSummaries[1])
	}

	writeReleasePayloadFile(t, defaultRoot, "skills/active/productivity/reviewer/SKILL.md", "post-update operator edit")
	rollback := RunUpdateReleaseAssetSkillRollback(context.Background(), UpdateReleaseAssetSkillRollbackOptions{
		SnapshotPath: snapshotPath,
	})
	if rollback.Failed {
		t.Fatalf("rollback failed: %+v", rollback)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: rollback.Evidence}, UpdateEvidenceReleaseAssetRollbackConflict)
	assertFileBody(t, defaultTarget, "post-update operator edit")
	assertFileBody(t, workTarget, "operator edited reviewer")
	if _, err := os.Stat(conflictPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rollback should remove unchanged conflict copy; stat err=%v", err)
	}
}

func writeReleasePayloadFile(t *testing.T, root, rel, body string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func releaseTestSHA256(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func releaseSkillDoc(name, description string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\nreview_state: reviewed\n---\n\n" + description + "."
}
