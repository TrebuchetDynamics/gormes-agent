package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateReleaseBinaryVerifiedSwapCreatesSnapshot(t *testing.T) {
	root := t.TempDir()
	managed := writeExecutableFile(t, filepath.Join(root, "home", "bin", "gormes"), "old-managed")
	published := writeExecutableFile(t, filepath.Join(root, "published", "gormes"), "old-published")
	plan := testReleaseBinaryPlan(filepath.Join(root, "snapshots", "20260516-010203-pre-update"))
	downloader := fakeReleaseDownloader("new-binary", "")

	report := RunUpdateReleaseBinaryUpdate(context.Background(), UpdateReleaseBinaryOptions{
		Plan:             plan,
		ManagedBinPath:   managed,
		PublishedBinPath: published,
		Downloader:       downloader,
		Runner:           fakeReleaseSmokeRunner{version: "0.2.13", commit: "abc1234"},
	})

	if report.Failed {
		t.Fatalf("RunUpdateReleaseBinaryUpdate failed: %+v", report)
	}
	if report.SnapshotID == "" || report.SnapshotPath != plan.SnapshotPath {
		t.Fatalf("snapshot = id %q path %q, want planned snapshot", report.SnapshotID, report.SnapshotPath)
	}
	assertFileBody(t, managed, "new-binary")
	assertFileBody(t, published, "new-binary")
	assertFileBody(t, filepath.Join(report.SnapshotPath, "managed.bin"), "old-managed")
	assertFileBody(t, filepath.Join(report.SnapshotPath, "published.bin"), "old-published")
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseSnapshotCreated)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseSmokePassed)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseSwapCompleted)
}

func TestUpdateReleaseBinaryChecksumMismatchFailsBeforeMutationEvenWithForce(t *testing.T) {
	root := t.TempDir()
	managed := writeExecutableFile(t, filepath.Join(root, "home", "bin", "gormes"), "old-managed")
	published := writeExecutableFile(t, filepath.Join(root, "published", "gormes"), "old-published")
	plan := testReleaseBinaryPlan(filepath.Join(root, "snapshots", "20260516-010203-pre-update"))

	report := RunUpdateReleaseBinaryUpdate(context.Background(), UpdateReleaseBinaryOptions{
		Plan:             plan,
		ManagedBinPath:   managed,
		PublishedBinPath: published,
		Downloader:       fakeReleaseDownloader("new-binary", strings.Repeat("0", 64)),
		Runner:           fakeReleaseSmokeRunner{version: "0.2.13", commit: "abc1234"},
		Force:            true,
	})

	if !report.Failed {
		t.Fatalf("checksum mismatch should fail report: %+v", report)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseChecksumFailed)
	assertFileBody(t, managed, "old-managed")
	assertFileBody(t, published, "old-published")
	if _, err := os.Stat(plan.SnapshotPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("integrity failure must not create snapshot before mutation; stat err=%v", err)
	}
}

func TestUpdateReleaseBinaryRollbackRestoresManagedAfterPublishedSwapFailure(t *testing.T) {
	root := t.TempDir()
	managed := writeExecutableFile(t, filepath.Join(root, "home", "bin", "gormes"), "old-managed")
	published := filepath.Join(root, "published", "gormes")
	if err := os.MkdirAll(published, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := testReleaseBinaryPlan(filepath.Join(root, "snapshots", "20260516-010203-pre-update"))

	report := RunUpdateReleaseBinaryUpdate(context.Background(), UpdateReleaseBinaryOptions{
		Plan:             plan,
		ManagedBinPath:   managed,
		PublishedBinPath: published,
		Downloader:       fakeReleaseDownloader("new-binary", ""),
		Runner:           fakeReleaseSmokeRunner{version: "0.2.13", commit: "abc1234"},
	})

	if !report.Failed {
		t.Fatalf("published directory should fail report: %+v", report)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseRollbackCompleted)
	assertFileBody(t, managed, "old-managed")
	if info, err := os.Stat(published); err != nil || !info.IsDir() {
		t.Fatalf("published directory not preserved after rollback: info=%v err=%v", info, err)
	}
}

func TestUpdateReleaseRollbackRestoresSnapshot(t *testing.T) {
	root := t.TempDir()
	managed := writeExecutableFile(t, filepath.Join(root, "home", "bin", "gormes"), "old-managed")
	published := writeExecutableFile(t, filepath.Join(root, "published", "gormes"), "old-published")
	plan := testReleaseBinaryPlan(filepath.Join(root, "snapshots", "snap-restore"))
	report := RunUpdateReleaseBinaryUpdate(context.Background(), UpdateReleaseBinaryOptions{
		Plan:             plan,
		ManagedBinPath:   managed,
		PublishedBinPath: published,
		Downloader:       fakeReleaseDownloader("new-binary", ""),
		Runner:           fakeReleaseSmokeRunner{version: "0.2.13", commit: "abc1234"},
	})
	if report.Failed {
		t.Fatalf("setup update failed: %+v", report)
	}
	writeExecutableFile(t, managed, "broken-managed")
	writeExecutableFile(t, published, "broken-published")

	rollback := RunUpdateReleaseRollback(context.Background(), UpdateReleaseRollbackOptions{
		SnapshotPath: report.SnapshotPath,
	})

	if rollback.Failed {
		t.Fatalf("rollback failed: %+v", rollback)
	}
	assertFileBody(t, managed, "old-managed")
	assertFileBody(t, published, "old-published")
	assertUpdateEvidence(t, UpdateReport{Evidence: rollback.Evidence}, UpdateEvidenceReleaseRollbackCompleted)
}

func testReleaseBinaryPlan(snapshotPath string) UpdateReleasePlan {
	return UpdateReleasePlan{
		InstallKind:  UpdateInstallKindRelease,
		Source:       UpdateSourceGitHubRelease,
		Channel:      UpdateReleaseChannelStable,
		Current:      UpdateBuildIdentity{Version: "0.2.12", GitCommit: "old"},
		Target:       UpdateReleaseMetadata{Version: "0.2.13", Tag: "v0.2.13", GitCommit: "abc1234"},
		ArtifactName: "gormes-0.2.13-linux-amd64.tar.gz",
		SnapshotPath: snapshotPath,
		Components:   []UpdateReleaseComponent{UpdateReleaseComponentSnapshot, UpdateReleaseComponentBinary, UpdateReleaseComponentChecksum},
	}
}

func fakeReleaseDownloader(body, expected string) UpdateReleaseArtifactDownloader {
	return func(_ context.Context, plan UpdateReleasePlan, stageDir string) (UpdateReleaseArtifact, error) {
		staged := filepath.Join(stageDir, "gormes")
		if err := os.WriteFile(staged, []byte(body), 0o755); err != nil {
			return UpdateReleaseArtifact{}, err
		}
		sum := sha256.Sum256([]byte(body))
		actual := hex.EncodeToString(sum[:])
		if expected == "" {
			expected = actual
		}
		return UpdateReleaseArtifact{
			ArtifactName:     plan.ArtifactName,
			StagedBinaryPath: staged,
			ExpectedSHA256:   expected,
			ActualSHA256:     actual,
		}, nil
	}
}

type fakeReleaseSmokeRunner struct {
	version string
	commit  string
	err     error
}

func (r fakeReleaseSmokeRunner) RunCommand(context.Context, string, []string, string, ...string) UpdateCommandResult {
	if r.err != nil {
		return UpdateCommandResult{Err: r.err, Stderr: r.err.Error()}
	}
	return UpdateCommandResult{Stdout: `{"version":"` + r.version + `","git_commit":"` + r.commit + `"}`}
}

func writeExecutableFile(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertFileBody(t *testing.T, path, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(body) != want {
		t.Fatalf("%s body = %q, want %q", path, body, want)
	}
}
