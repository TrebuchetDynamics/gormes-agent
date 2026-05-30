package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	UpdateEvidenceReleaseDownloadCompleted  UpdateEvidenceKind = "update_release_download_completed"
	UpdateEvidenceReleaseChecksumVerified   UpdateEvidenceKind = "update_release_checksum_verified"
	UpdateEvidenceReleaseChecksumFailed     UpdateEvidenceKind = "update_release_checksum_failed"
	UpdateEvidenceReleaseProvenanceVerified UpdateEvidenceKind = "update_release_provenance_verified"
	UpdateEvidenceReleaseProvenanceFailed   UpdateEvidenceKind = "update_release_provenance_failed"
	UpdateEvidenceReleaseSnapshotCreated    UpdateEvidenceKind = "update_release_snapshot_created"
	UpdateEvidenceReleaseSmokePassed        UpdateEvidenceKind = "update_release_smoke_passed"
	UpdateEvidenceReleaseSmokeFailed        UpdateEvidenceKind = "update_release_smoke_failed"
	UpdateEvidenceReleaseSwapCompleted      UpdateEvidenceKind = "update_release_swap_completed"
	UpdateEvidenceReleaseSwapFailed         UpdateEvidenceKind = "update_release_swap_failed"
	UpdateEvidenceReleaseRollbackCompleted  UpdateEvidenceKind = "update_release_rollback_completed"
	UpdateEvidenceReleaseRollbackFailed     UpdateEvidenceKind = "update_release_rollback_failed"
)

type UpdateReleaseArtifact struct {
	ArtifactName     string
	ArtifactPath     string
	StagedBinaryPath string
	ExpectedSHA256   string
	ActualSHA256     string
}

type UpdateReleaseArtifactDownloader func(context.Context, UpdateReleasePlan, string) (UpdateReleaseArtifact, error)

type UpdateReleaseProvenanceVerifier func(context.Context, UpdateReleaseArtifact, UpdateReleasePlan) error

type UpdateReleaseBinaryOptions struct {
	Plan               UpdateReleasePlan
	ManagedBinPath     string
	PublishedBinPath   string
	Downloader         UpdateReleaseArtifactDownloader
	Runner             UpdateCommandRunner
	ProvenanceVerifier UpdateReleaseProvenanceVerifier
	Force              bool
}

type UpdateReleaseRollbackOptions struct {
	SnapshotID   string
	SnapshotRoot string
	SnapshotPath string
}

type UpdateReleaseBinaryReport struct {
	Failed           bool
	SnapshotID       string
	SnapshotPath     string
	PreviousVersion  string
	NewVersion       string
	ManagedBinPath   string
	PublishedBinPath string
	Evidence         []UpdateEvidence
	OperatorRecovery string
}

func (r *UpdateReleaseBinaryReport) add(kind UpdateEvidenceKind, detail string) {
	r.Evidence = append(r.Evidence, UpdateEvidence{Kind: kind, Detail: detail})
}

func RunUpdateReleaseBinaryUpdate(ctx context.Context, opts UpdateReleaseBinaryOptions) UpdateReleaseBinaryReport {
	report := UpdateReleaseBinaryReport{
		PreviousVersion:  opts.Plan.Current.Version,
		NewVersion:       opts.Plan.Target.Version,
		ManagedBinPath:   strings.TrimSpace(opts.ManagedBinPath),
		PublishedBinPath: strings.TrimSpace(opts.PublishedBinPath),
	}
	if opts.Plan.Source != UpdateSourceGitHubRelease {
		report.Failed = true
		report.add(UpdateEvidenceReleaseDownloadCompleted, "release binary update requires a github_release plan")
		return report
	}
	if opts.Plan.ArtifactName == "" {
		report.Failed = true
		report.add(UpdateEvidenceReleaseDownloadCompleted, "release plan is missing an artifact name")
		return report
	}
	if report.ManagedBinPath == "" || report.PublishedBinPath == "" {
		report.Failed = true
		report.add(UpdateEvidenceReleaseSwapFailed, "missing managed or published binary path")
		return report
	}
	downloader := opts.Downloader
	if downloader == nil {
		downloader = defaultUpdateReleaseArtifactDownloader
	}
	runner := opts.Runner
	if runner == nil {
		runner = RealUpdateCommandRunner{}
	}
	stageDir, err := os.MkdirTemp(filepath.Dir(report.ManagedBinPath), ".gormes-release-stage-*")
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseDownloadCompleted, err.Error())
		return report
	}
	defer os.RemoveAll(stageDir)

	artifact, err := downloader(ctx, opts.Plan, stageDir)
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseDownloadCompleted, err.Error())
		return report
	}
	if artifact.ArtifactName != opts.Plan.ArtifactName {
		report.Failed = true
		report.add(UpdateEvidenceReleaseDownloadCompleted, fmt.Sprintf("artifact mismatch: got %q want %q", artifact.ArtifactName, opts.Plan.ArtifactName))
		return report
	}
	report.add(UpdateEvidenceReleaseDownloadCompleted, artifact.ArtifactName)
	if err := verifyReleaseArtifactChecksum(artifact); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseChecksumFailed, err.Error())
		return report
	}
	report.add(UpdateEvidenceReleaseChecksumVerified, shortChecksumEvidence(artifact.ExpectedSHA256))
	if opts.ProvenanceVerifier != nil {
		if err := opts.ProvenanceVerifier(ctx, artifact, opts.Plan); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseProvenanceFailed, err.Error())
			return report
		}
		report.add(UpdateEvidenceReleaseProvenanceVerified, "provenance verified")
	}
	if err := smokeUpdateReleaseBinary(ctx, runner, artifact.StagedBinaryPath, opts.Plan); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseSmokeFailed, err.Error())
		return report
	}
	report.add(UpdateEvidenceReleaseSmokePassed, fmt.Sprintf("%s (%s)", opts.Plan.Target.Version, opts.Plan.Target.GitCommit))

	snapshot, err := createUpdateReleaseSnapshot(ctx, opts.Plan.SnapshotPath, report.ManagedBinPath, report.PublishedBinPath, opts.Plan)
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseSnapshotCreated, err.Error())
		return report
	}
	report.SnapshotID = snapshot.ID
	report.SnapshotPath = snapshot.Path
	report.add(UpdateEvidenceReleaseSnapshotCreated, snapshot.Path)

	if err := replaceBinaryAtomically(artifact.StagedBinaryPath, report.ManagedBinPath); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseSwapFailed, fmt.Sprintf("managed binary: %v", err))
		rollbackReleaseSnapshotIntoReport(ctx, &report, snapshot.Path)
		return report
	}
	if err := replaceBinaryAtomically(artifact.StagedBinaryPath, report.PublishedBinPath); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseSwapFailed, fmt.Sprintf("published binary: %v", err))
		rollbackReleaseSnapshotIntoReport(ctx, &report, snapshot.Path)
		return report
	}
	report.add(UpdateEvidenceReleaseSwapCompleted, fmt.Sprintf("%s -> %s", report.PreviousVersion, report.NewVersion))
	return report
}

func RunUpdateReleaseRollback(ctx context.Context, opts UpdateReleaseRollbackOptions) UpdateReleaseBinaryReport {
	report := UpdateReleaseBinaryReport{}
	snapshotPath, err := resolveUpdateReleaseSnapshotPath(opts)
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseRollbackFailed, err.Error())
		return report
	}
	report.SnapshotPath = snapshotPath
	report.SnapshotID = filepath.Base(snapshotPath)
	manifest, err := readUpdateReleaseSnapshotManifest(snapshotPath)
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseRollbackFailed, err.Error())
		return report
	}
	report.ManagedBinPath = manifest.ManagedBinPath
	report.PublishedBinPath = manifest.PublishedBinPath
	report.PreviousVersion = manifest.TargetVersion
	report.NewVersion = manifest.PreviousVersion
	if err := restoreUpdateReleaseSnapshot(ctx, snapshotPath, manifest); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseRollbackFailed, err.Error())
		return report
	}
	report.add(UpdateEvidenceReleaseRollbackCompleted, snapshotPath)
	return report
}

func rollbackReleaseSnapshotIntoReport(ctx context.Context, report *UpdateReleaseBinaryReport, snapshotPath string) {
	rollback := RunUpdateReleaseRollback(ctx, UpdateReleaseRollbackOptions{SnapshotPath: snapshotPath})
	report.Evidence = append(report.Evidence, rollback.Evidence...)
	if rollback.Failed {
		report.OperatorRecovery = fmt.Sprintf("manual restore needed from %s", snapshotPath)
		return
	}
	report.OperatorRecovery = fmt.Sprintf("rolled back from %s", snapshotPath)
}

func verifyReleaseArtifactChecksum(artifact UpdateReleaseArtifact) error {
	expected := strings.ToLower(strings.TrimSpace(artifact.ExpectedSHA256))
	if expected == "" {
		return fmt.Errorf("missing SHA-256 checksum")
	}
	actual := strings.ToLower(strings.TrimSpace(artifact.ActualSHA256))
	if actual == "" {
		var err error
		actual, err = fileSHA256(artifact.StagedBinaryPath)
		if err != nil {
			return fmt.Errorf("compute SHA-256: %w", err)
		}
	}
	if expected != actual {
		return fmt.Errorf("SHA-256 mismatch for %s: expected %s got %s", artifact.ArtifactName, expected, actual)
	}
	return nil
}

func shortChecksumEvidence(sum string) string {
	sum = strings.TrimSpace(sum)
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}

func smokeUpdateReleaseBinary(ctx context.Context, runner UpdateCommandRunner, binaryPath string, plan UpdateReleasePlan) error {
	result := runner.RunCommand(ctx, "", nil, binaryPath, "version", "--json")
	if result.Err != nil {
		return fmt.Errorf("version --json failed: %s", commandFailureDetail(result))
	}
	var got struct {
		Version   string `json:"version"`
		GitCommit string `json:"git_commit"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		return fmt.Errorf("version --json was not valid JSON: %w", err)
	}
	if strings.TrimSpace(got.Version) != strings.TrimSpace(plan.Target.Version) {
		return fmt.Errorf("version mismatch: got %s want %s", got.Version, plan.Target.Version)
	}
	if comparableReleaseCommit(plan.Target.GitCommit) && strings.TrimSpace(got.GitCommit) != strings.TrimSpace(plan.Target.GitCommit) {
		return fmt.Errorf("git_commit mismatch: got %s want %s", got.GitCommit, plan.Target.GitCommit)
	}
	return nil
}

var updateReleaseComparableCommitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

func comparableReleaseCommit(commit string) bool {
	return updateReleaseComparableCommitPattern.MatchString(strings.TrimSpace(commit))
}

type updateReleaseSnapshotManifest struct {
	ID               string `json:"id"`
	CreatedAt        string `json:"created_at"`
	ManagedBinPath   string `json:"managed_bin_path"`
	PublishedBinPath string `json:"published_bin_path"`
	ManagedExisted   bool   `json:"managed_existed"`
	PublishedExisted bool   `json:"published_existed"`
	ManagedWasDir    bool   `json:"managed_was_dir,omitempty"`
	PublishedWasDir  bool   `json:"published_was_dir,omitempty"`
	PreviousVersion  string `json:"previous_version"`
	TargetVersion    string `json:"target_version"`
}

type updateReleaseSnapshot struct {
	ID   string
	Path string
}

func createUpdateReleaseSnapshot(ctx context.Context, snapshotPath, managedBin, publishedBin string, plan UpdateReleasePlan) (updateReleaseSnapshot, error) {
	if strings.TrimSpace(snapshotPath) == "" {
		return updateReleaseSnapshot{}, fmt.Errorf("release snapshot path is empty")
	}
	id := filepath.Base(snapshotPath)
	if err := os.MkdirAll(snapshotPath, 0o755); err != nil {
		return updateReleaseSnapshot{}, err
	}
	manifest := updateReleaseSnapshotManifest{
		ID:               id,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		ManagedBinPath:   managedBin,
		PublishedBinPath: publishedBin,
		PreviousVersion:  plan.Current.Version,
		TargetVersion:    plan.Target.Version,
	}
	var err error
	manifest.ManagedExisted, manifest.ManagedWasDir, err = snapshotReleaseBinaryPath(ctx, managedBin, filepath.Join(snapshotPath, "managed.bin"))
	if err != nil {
		return updateReleaseSnapshot{}, fmt.Errorf("snapshot managed binary: %w", err)
	}
	manifest.PublishedExisted, manifest.PublishedWasDir, err = snapshotReleaseBinaryPath(ctx, publishedBin, filepath.Join(snapshotPath, "published.bin"))
	if err != nil {
		return updateReleaseSnapshot{}, fmt.Errorf("snapshot published binary: %w", err)
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return updateReleaseSnapshot{}, err
	}
	if err := os.WriteFile(filepath.Join(snapshotPath, "manifest.json"), body, 0o644); err != nil {
		return updateReleaseSnapshot{}, err
	}
	return updateReleaseSnapshot{ID: id, Path: snapshotPath}, nil
}

func snapshotReleaseBinaryPath(ctx context.Context, source, dest string) (existed bool, wasDir bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, false, err
	}
	info, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	if info.IsDir() {
		return true, true, nil
	}
	if err := copyFile(source, dest); err != nil {
		return false, false, err
	}
	return true, false, nil
}

func readUpdateReleaseSnapshotManifest(snapshotPath string) (updateReleaseSnapshotManifest, error) {
	body, err := os.ReadFile(filepath.Join(snapshotPath, "manifest.json"))
	if err != nil {
		return updateReleaseSnapshotManifest{}, err
	}
	var manifest updateReleaseSnapshotManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return updateReleaseSnapshotManifest{}, err
	}
	return manifest, nil
}

func restoreUpdateReleaseSnapshot(ctx context.Context, snapshotPath string, manifest updateReleaseSnapshotManifest) error {
	if err := restoreUpdateReleaseSnapshotPath(ctx, filepath.Join(snapshotPath, "managed.bin"), manifest.ManagedBinPath, manifest.ManagedExisted, manifest.ManagedWasDir); err != nil {
		return fmt.Errorf("restore managed binary: %w", err)
	}
	if err := restoreUpdateReleaseSnapshotPath(ctx, filepath.Join(snapshotPath, "published.bin"), manifest.PublishedBinPath, manifest.PublishedExisted, manifest.PublishedWasDir); err != nil {
		return fmt.Errorf("restore published binary: %w", err)
	}
	return nil
}

func restoreUpdateReleaseSnapshotPath(ctx context.Context, source, target string, existed, wasDir bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if wasDir {
		return nil
	}
	if !existed {
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := replaceBinaryAtomically(source, target); err != nil {
		return err
	}
	return nil
}

func resolveUpdateReleaseSnapshotPath(opts UpdateReleaseRollbackOptions) (string, error) {
	if strings.TrimSpace(opts.SnapshotPath) != "" {
		return opts.SnapshotPath, nil
	}
	id := strings.TrimSpace(opts.SnapshotID)
	if id == "" {
		return "", fmt.Errorf("missing release update snapshot id")
	}
	if filepath.Base(id) != id {
		return "", fmt.Errorf("invalid release update snapshot id %q", id)
	}
	root := strings.TrimSpace(opts.SnapshotRoot)
	if root == "" {
		return "", fmt.Errorf("missing release update snapshot root")
	}
	return filepath.Join(root, id), nil
}

func replaceBinaryAtomically(source, target string) error {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
		return fmt.Errorf("source and target are required")
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return fmt.Errorf("cannot replace directory %s", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", target, os.Getpid())
	_ = os.Remove(tmp)
	if err := copyFile(source, tmp); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func defaultUpdateReleaseArtifactDownloader(ctx context.Context, plan UpdateReleasePlan, stageDir string) (UpdateReleaseArtifact, error) {
	base := strings.TrimSpace(os.Getenv("GORMES_RELEASES_DOWNLOAD_BASE"))
	if base == "" {
		base = "https://github.com/TrebuchetDynamics/gormes-agent/releases/download"
	}
	tag := strings.TrimSpace(plan.Target.Tag)
	if tag == "" {
		tag = "v" + strings.TrimPrefix(plan.Target.Version, "v")
	}
	if tag == "v" || plan.ArtifactName == "" {
		return UpdateReleaseArtifact{}, fmt.Errorf("release plan missing tag or artifact")
	}
	artifactURL := strings.TrimRight(base, "/") + "/" + tag + "/" + plan.ArtifactName
	artifactPath := filepath.Join(stageDir, plan.ArtifactName)
	if err := downloadUpdateReleaseFile(ctx, artifactURL, artifactPath); err != nil {
		return UpdateReleaseArtifact{}, err
	}
	shaPath := artifactPath + ".sha256"
	if err := downloadUpdateReleaseFile(ctx, artifactURL+".sha256", shaPath); err != nil {
		return UpdateReleaseArtifact{}, err
	}
	expected, err := readUpdateReleaseChecksumFile(shaPath)
	if err != nil {
		return UpdateReleaseArtifact{}, err
	}
	actual, err := fileSHA256(artifactPath)
	if err != nil {
		return UpdateReleaseArtifact{}, err
	}
	if strings.ToLower(expected) != strings.ToLower(actual) {
		return UpdateReleaseArtifact{}, fmt.Errorf("SHA-256 mismatch for %s: expected %s got %s", plan.ArtifactName, expected, actual)
	}
	stagedBinary, err := extractUpdateReleaseBinary(artifactPath, stageDir)
	if err != nil {
		return UpdateReleaseArtifact{}, err
	}
	return UpdateReleaseArtifact{
		ArtifactName:     plan.ArtifactName,
		ArtifactPath:     artifactPath,
		StagedBinaryPath: stagedBinary,
		ExpectedSHA256:   expected,
		ActualSHA256:     actual,
	}, nil
}

func downloadUpdateReleaseFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download %s: HTTP %s", url, resp.Status)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func readUpdateReleaseChecksumFile(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty SHA-256 sidecar")
	}
	return fields[0], nil
}

func extractUpdateReleaseBinary(artifactPath, stageDir string) (string, error) {
	file, err := os.Open(artifactPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(header.Name)
		if name != "gormes" && name != "gormes.exe" {
			continue
		}
		target := filepath.Join(stageDir, name)
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return target, nil
	}
	return "", fmt.Errorf("release artifact does not contain a gormes binary")
}
