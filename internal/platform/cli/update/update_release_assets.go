package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/pathguard"
)

const (
	UpdateEvidenceReleaseManifestVerified       UpdateEvidenceKind = "update_release_manifest_verified"
	UpdateEvidenceReleaseManifestFailed         UpdateEvidenceKind = "update_release_manifest_failed"
	UpdateEvidenceReleaseAssetSyncCompleted     UpdateEvidenceKind = "update_release_asset_sync_completed"
	UpdateEvidenceReleaseAssetSyncFailed        UpdateEvidenceKind = "update_release_asset_sync_failed"
	UpdateEvidenceReleaseSkillSyncCompleted     UpdateEvidenceKind = "update_release_skill_sync_completed"
	UpdateEvidenceReleaseSkillSyncFailed        UpdateEvidenceKind = "update_release_skill_sync_failed"
	UpdateEvidenceReleaseAssetRollbackCompleted UpdateEvidenceKind = "update_release_asset_rollback_completed"
	UpdateEvidenceReleaseAssetRollbackFailed    UpdateEvidenceKind = "update_release_asset_rollback_failed"
	UpdateEvidenceReleaseAssetRollbackConflict  UpdateEvidenceKind = "update_release_asset_rollback_conflict"
)

const updateReleaseManifestSchemaVersion = 1

type UpdateReleaseManifest struct {
	SchemaVersion int                               `json:"schema_version"`
	Release       UpdateReleaseMetadata             `json:"release,omitempty"`
	Assets        []UpdateReleaseAssetManifestEntry `json:"assets,omitempty"`
	Skills        []UpdateReleaseSkillManifestEntry `json:"skills,omitempty"`
}

type UpdateReleaseAssetManifestEntry struct {
	Path        string `json:"path"`
	PayloadPath string `json:"payload_path"`
	SHA256      string `json:"sha256"`
}

type UpdateReleaseSkillManifestEntry struct {
	Name           string `json:"name"`
	Profile        string `json:"profile,omitempty"`
	Path           string `json:"path"`
	PayloadPath    string `json:"payload_path,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	PreviousSHA256 string `json:"previous_sha256,omitempty"`
	Removed        bool   `json:"removed,omitempty"`
}

type UpdateReleaseAssetSkillSyncOptions struct {
	Plan          UpdateReleasePlan
	Manifest      UpdateReleaseManifest
	PayloadRoot   string
	AssetRoot     string
	SnapshotPath  string
	SkillProfiles []skills.SkillProfileRoot
}

type UpdateReleaseAssetSkillRollbackOptions struct {
	SnapshotPath string
}

type UpdateReleaseAssetSkillSyncReport struct {
	Failed         bool
	SnapshotID     string
	SnapshotPath   string
	Evidence       []UpdateEvidence
	SkillSummaries []skills.SkillProfileSyncSummary
}

func (r *UpdateReleaseAssetSkillSyncReport) add(kind UpdateEvidenceKind, detail string) {
	r.Evidence = append(r.Evidence, UpdateEvidence{Kind: kind, Detail: detail})
}

func RunUpdateReleaseAssetSkillSync(ctx context.Context, opts UpdateReleaseAssetSkillSyncOptions) UpdateReleaseAssetSkillSyncReport {
	report := UpdateReleaseAssetSkillSyncReport{SnapshotPath: strings.TrimSpace(opts.SnapshotPath)}
	if err := ctx.Err(); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseManifestFailed, err.Error())
		return report
	}
	if report.SnapshotPath == "" {
		report.SnapshotPath = opts.Plan.SnapshotPath
	}
	if err := validateUpdateReleaseManifest(opts); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseManifestFailed, err.Error())
		return report
	}
	report.add(UpdateEvidenceReleaseManifestVerified, "schema_version=1")
	if len(opts.Manifest.Assets) == 0 && len(opts.Manifest.Skills) == 0 {
		return report
	}

	snapshot, err := createUpdateReleaseAssetSnapshot(ctx, report.SnapshotPath, opts)
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseAssetSyncFailed, err.Error())
		return report
	}
	report.SnapshotID = snapshot.ID
	report.SnapshotPath = snapshot.Path
	manifest := snapshot.Manifest

	for _, asset := range opts.Manifest.Assets {
		source, err := updateReleasePayloadPath(opts.PayloadRoot, asset.PayloadPath)
		if err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseAssetSyncFailed, err.Error())
			rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
			return report
		}
		target, err := updateReleaseTargetPath(opts.AssetRoot, asset.Path)
		if err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseAssetSyncFailed, err.Error())
			rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
			return report
		}
		if err := replaceReleaseDataFile(source, target); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseAssetSyncFailed, err.Error())
			rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
			return report
		}
		if sum, ok, err := updateReleaseFileSHAIfExists(target); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseAssetSyncFailed, err.Error())
			rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
			return report
		} else {
			setUpdateReleaseAssetSnapshotAfter(&manifest, target, ok, sum)
		}
	}
	if len(opts.Manifest.Assets) > 0 {
		report.add(UpdateEvidenceReleaseAssetSyncCompleted, fmt.Sprintf("assets updated=%d", len(opts.Manifest.Assets)))
	}

	if len(opts.Manifest.Skills) > 0 {
		skillReport, err := skills.SyncBundledSkillsFromManifest(ctx, skills.BundledSkillManifestSyncRequest{
			PayloadRoot: opts.PayloadRoot,
			Profiles:    opts.SkillProfiles,
			Entries:     updateReleaseSkillEntriesForSkillsPackage(opts.Manifest.Skills),
		})
		if err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseSkillSyncFailed, err.Error())
			rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
			return report
		}
		report.SkillSummaries = skillReport.Summaries
		report.Evidence = append(report.Evidence, skillEvidenceForReleaseReport(skillReport.Evidence)...)
		for _, target := range updateReleaseSkillSnapshotTargets(opts) {
			if sum, ok, err := updateReleaseFileSHAIfExists(target); err != nil {
				report.Failed = true
				report.add(UpdateEvidenceReleaseSkillSyncFailed, err.Error())
				rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
				return report
			} else {
				setUpdateReleaseAssetSnapshotAfter(&manifest, target, ok, sum)
			}
		}
		report.add(UpdateEvidenceReleaseSkillSyncCompleted, formatReleaseSkillSummaries(skillReport.Summaries))
	}

	if err := writeUpdateReleaseAssetSnapshotManifest(snapshot.Path, manifest); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseAssetSyncFailed, err.Error())
		rollbackUpdateReleaseAssetSnapshotIntoReport(ctx, &report, snapshot.Path)
		return report
	}
	return report
}

func RunUpdateReleaseAssetSkillRollback(ctx context.Context, opts UpdateReleaseAssetSkillRollbackOptions) UpdateReleaseAssetSkillSyncReport {
	report := UpdateReleaseAssetSkillSyncReport{SnapshotPath: strings.TrimSpace(opts.SnapshotPath)}
	if report.SnapshotPath == "" {
		report.Failed = true
		report.add(UpdateEvidenceReleaseAssetRollbackFailed, "missing release asset snapshot path")
		return report
	}
	report.SnapshotID = filepath.Base(report.SnapshotPath)
	manifest, err := readUpdateReleaseAssetSnapshotManifest(report.SnapshotPath)
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseAssetRollbackFailed, err.Error())
		return report
	}
	for _, file := range manifest.Files {
		if err := ctx.Err(); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseAssetRollbackFailed, err.Error())
			return report
		}
		if updateReleaseSnapshotFileEditedAfterUpdate(file) {
			report.add(UpdateEvidenceReleaseAssetRollbackConflict, file.TargetPath)
			continue
		}
		if err := restoreUpdateReleaseAssetSnapshotFile(report.SnapshotPath, file); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseAssetRollbackFailed, err.Error())
			return report
		}
	}
	report.add(UpdateEvidenceReleaseAssetRollbackCompleted, report.SnapshotPath)
	return report
}

func validateUpdateReleaseManifest(opts UpdateReleaseAssetSkillSyncOptions) error {
	manifest := opts.Manifest
	if manifest.SchemaVersion != updateReleaseManifestSchemaVersion {
		return fmt.Errorf("unsupported release manifest schema_version %d", manifest.SchemaVersion)
	}
	seenAssets := map[string]bool{}
	for _, asset := range manifest.Assets {
		targetRel, err := cleanUpdateReleaseManifestPath(asset.Path)
		if err != nil {
			return fmt.Errorf("asset path %q: %w", asset.Path, err)
		}
		if seenAssets[targetRel] {
			return fmt.Errorf("duplicate asset path %q", targetRel)
		}
		seenAssets[targetRel] = true
		if _, err := cleanUpdateReleaseManifestPath(asset.PayloadPath); err != nil {
			return fmt.Errorf("asset payload_path %q: %w", asset.PayloadPath, err)
		}
		expected := strings.ToLower(strings.TrimSpace(asset.SHA256))
		if expected == "" {
			return fmt.Errorf("asset %q missing sha256", targetRel)
		}
		payload, err := updateReleasePayloadPath(opts.PayloadRoot, asset.PayloadPath)
		if err != nil {
			return fmt.Errorf("asset %q payload: %w", targetRel, err)
		}
		actual, err := fileSHA256(payload)
		if err != nil {
			return fmt.Errorf("asset %q checksum: %w", targetRel, err)
		}
		if actual != expected {
			return fmt.Errorf("asset %q SHA-256 mismatch: expected %s got %s", targetRel, expected, actual)
		}
	}
	seenSkills := map[string]bool{}
	for _, skill := range manifest.Skills {
		targetRel, err := cleanUpdateReleaseManifestPath(skill.Path)
		if err != nil {
			return fmt.Errorf("skill path %q: %w", skill.Path, err)
		}
		if filepath.Base(targetRel) != "SKILL.md" {
			return fmt.Errorf("skill path %q must end in SKILL.md", targetRel)
		}
		profile := strings.TrimSpace(skill.Profile)
		key := profile + "\x00" + targetRel
		if seenSkills[key] {
			return fmt.Errorf("duplicate skill path %q for profile %q", targetRel, profile)
		}
		seenSkills[key] = true
		if !skill.Removed {
			if _, err := cleanUpdateReleaseManifestPath(skill.PayloadPath); err != nil {
				return fmt.Errorf("skill payload_path %q: %w", skill.PayloadPath, err)
			}
			expected := strings.ToLower(strings.TrimSpace(skill.SHA256))
			if expected == "" {
				return fmt.Errorf("skill %q missing sha256", targetRel)
			}
			payload, err := updateReleasePayloadPath(opts.PayloadRoot, skill.PayloadPath)
			if err != nil {
				return fmt.Errorf("skill %q payload: %w", targetRel, err)
			}
			actual, err := fileSHA256(payload)
			if err != nil {
				return fmt.Errorf("skill %q checksum: %w", targetRel, err)
			}
			if actual != expected {
				return fmt.Errorf("skill %q SHA-256 mismatch: expected %s got %s", targetRel, expected, actual)
			}
		}
	}
	return nil
}

func cleanUpdateReleaseManifestPath(path string) (string, error) {
	return pathguard.CleanRelative(path)
}

type updateReleaseAssetSnapshot struct {
	ID       string
	Path     string
	Manifest updateReleaseAssetSnapshotManifest
}

type updateReleaseAssetSnapshotManifest struct {
	ID        string                           `json:"id"`
	CreatedAt string                           `json:"created_at"`
	Files     []updateReleaseAssetSnapshotFile `json:"files"`
}

type updateReleaseAssetSnapshotFile struct {
	TargetPath    string `json:"target_path"`
	SnapshotPath  string `json:"snapshot_path,omitempty"`
	ExistedBefore bool   `json:"existed_before"`
	BeforeSHA256  string `json:"before_sha256,omitempty"`
	ExistedAfter  bool   `json:"existed_after"`
	AfterSHA256   string `json:"after_sha256,omitempty"`
}

func createUpdateReleaseAssetSnapshot(ctx context.Context, snapshotPath string, opts UpdateReleaseAssetSkillSyncOptions) (updateReleaseAssetSnapshot, error) {
	if strings.TrimSpace(snapshotPath) == "" {
		return updateReleaseAssetSnapshot{}, fmt.Errorf("release asset snapshot path is empty")
	}
	if err := os.MkdirAll(snapshotPath, 0o755); err != nil {
		return updateReleaseAssetSnapshot{}, err
	}
	targets := updateReleaseSnapshotTargetPaths(opts)
	manifest := updateReleaseAssetSnapshotManifest{
		ID:        filepath.Base(snapshotPath),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Files:     make([]updateReleaseAssetSnapshotFile, 0, len(targets)),
	}
	for i, target := range targets {
		if err := ctx.Err(); err != nil {
			return updateReleaseAssetSnapshot{}, err
		}
		record := updateReleaseAssetSnapshotFile{TargetPath: target}
		if info, err := os.Stat(target); err == nil {
			if info.IsDir() {
				return updateReleaseAssetSnapshot{}, fmt.Errorf("cannot snapshot directory %s", target)
			}
			record.ExistedBefore = true
			record.SnapshotPath = filepath.Join("files", fmt.Sprintf("%04d", i), filepath.Base(target))
			if err := os.MkdirAll(filepath.Join(snapshotPath, filepath.Dir(record.SnapshotPath)), 0o755); err != nil {
				return updateReleaseAssetSnapshot{}, err
			}
			if err := copyFile(target, filepath.Join(snapshotPath, record.SnapshotPath)); err != nil {
				return updateReleaseAssetSnapshot{}, err
			}
			sum, err := fileSHA256(target)
			if err != nil {
				return updateReleaseAssetSnapshot{}, err
			}
			record.BeforeSHA256 = sum
		} else if !os.IsNotExist(err) {
			return updateReleaseAssetSnapshot{}, err
		}
		manifest.Files = append(manifest.Files, record)
	}
	if err := writeUpdateReleaseAssetSnapshotManifest(snapshotPath, manifest); err != nil {
		return updateReleaseAssetSnapshot{}, err
	}
	return updateReleaseAssetSnapshot{ID: manifest.ID, Path: snapshotPath, Manifest: manifest}, nil
}

func updateReleaseSnapshotTargetPaths(opts UpdateReleaseAssetSkillSyncOptions) []string {
	seen := map[string]bool{}
	var out []string
	for _, asset := range opts.Manifest.Assets {
		target, err := updateReleaseTargetPath(opts.AssetRoot, asset.Path)
		if err == nil && !seen[target] {
			seen[target] = true
			out = append(out, target)
		}
	}
	for _, target := range updateReleaseSkillSnapshotTargets(opts) {
		if !seen[target] {
			seen[target] = true
			out = append(out, target)
		}
	}
	sort.Strings(out)
	return out
}

func updateReleaseSkillSnapshotTargets(opts UpdateReleaseAssetSkillSyncOptions) []string {
	var out []string
	for _, profile := range opts.SkillProfiles {
		if strings.TrimSpace(profile.Root) == "" {
			continue
		}
		for _, entry := range opts.Manifest.Skills {
			if !releaseSkillEntryAppliesToProfile(entry, profile.Name) {
				continue
			}
			rel, err := cleanUpdateReleaseManifestPath(entry.Path)
			if err != nil {
				continue
			}
			active := filepath.Join(profile.Root, "skills", "active", rel)
			if updateReleasePathWithin(profile.Root, active) {
				out = append(out, active)
			}
			if !entry.Removed {
				conflict := filepath.Join(profile.Root, "skills", ".bundled-conflicts", releaseSkillEntryName(entry, rel), shortReleaseDigest(entry.SHA256), rel)
				if updateReleasePathWithin(profile.Root, conflict) {
					out = append(out, conflict)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

func releaseSkillEntryAppliesToProfile(entry UpdateReleaseSkillManifestEntry, profile string) bool {
	want := strings.TrimSpace(entry.Profile)
	return want == "" || want == "*" || want == profile
}

func releaseSkillEntryName(entry UpdateReleaseSkillManifestEntry, rel string) string {
	name := strings.TrimSpace(entry.Name)
	if name != "" {
		return name
	}
	return filepath.Base(filepath.Dir(rel))
}

func shortReleaseDigest(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) > 12 {
		return value[:12]
	}
	if value == "" {
		return "unknown"
	}
	return value
}

func setUpdateReleaseAssetSnapshotAfter(manifest *updateReleaseAssetSnapshotManifest, target string, existed bool, sha string) {
	for i := range manifest.Files {
		if manifest.Files[i].TargetPath == target {
			manifest.Files[i].ExistedAfter = existed
			manifest.Files[i].AfterSHA256 = sha
			return
		}
	}
}

func updateReleaseFileSHAIfExists(path string) (sha string, existed bool, err error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return "", false, err
	}
	return sum, true, nil
}

func writeUpdateReleaseAssetSnapshotManifest(snapshotPath string, manifest updateReleaseAssetSnapshotManifest) error {
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(snapshotPath, "manifest.json"), body, 0o644)
}

func readUpdateReleaseAssetSnapshotManifest(snapshotPath string) (updateReleaseAssetSnapshotManifest, error) {
	body, err := os.ReadFile(filepath.Join(snapshotPath, "manifest.json"))
	if err != nil {
		return updateReleaseAssetSnapshotManifest{}, err
	}
	var manifest updateReleaseAssetSnapshotManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return updateReleaseAssetSnapshotManifest{}, err
	}
	return manifest, nil
}

func restoreUpdateReleaseAssetSnapshotFile(snapshotPath string, file updateReleaseAssetSnapshotFile) error {
	if file.ExistedBefore {
		return replaceReleaseDataFile(filepath.Join(snapshotPath, file.SnapshotPath), file.TargetPath)
	}
	if err := os.Remove(file.TargetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func updateReleaseSnapshotFileEditedAfterUpdate(file updateReleaseAssetSnapshotFile) bool {
	_, err := os.Stat(file.TargetPath)
	if os.IsNotExist(err) {
		return file.ExistedAfter
	}
	if err != nil {
		return true
	}
	current, err := fileSHA256(file.TargetPath)
	if err != nil {
		return true
	}
	return current != strings.ToLower(strings.TrimSpace(file.AfterSHA256))
}

func rollbackUpdateReleaseAssetSnapshotIntoReport(ctx context.Context, report *UpdateReleaseAssetSkillSyncReport, snapshotPath string) {
	rollback := RunUpdateReleaseAssetSkillRollback(ctx, UpdateReleaseAssetSkillRollbackOptions{SnapshotPath: snapshotPath})
	report.Evidence = append(report.Evidence, rollback.Evidence...)
}

func updateReleasePayloadPath(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("payload root is empty")
	}
	clean, err := cleanUpdateReleaseManifestPath(rel)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, clean)
	if !updateReleasePathWithin(root, path) {
		return "", fmt.Errorf("payload path escapes root")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("payload symlink is not allowed: %s", clean)
	}
	if info.IsDir() {
		return "", fmt.Errorf("payload directory is not allowed: %s", clean)
	}
	return path, nil
}

func updateReleaseTargetPath(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("target root is empty")
	}
	clean, err := cleanUpdateReleaseManifestPath(rel)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, clean)
	if !updateReleasePathWithin(root, path) {
		return "", fmt.Errorf("target path escapes root")
	}
	return path, nil
}

func updateReleasePathWithin(root, target string) bool {
	return pathguard.Within(root, target)
}

func replaceReleaseDataFile(source, target string) error {
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
	if err := os.Chmod(tmp, 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func updateReleaseSkillEntriesForSkillsPackage(entries []UpdateReleaseSkillManifestEntry) []skills.BundledSkillManifestEntry {
	out := make([]skills.BundledSkillManifestEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, skills.BundledSkillManifestEntry{
			Name:           entry.Name,
			Profile:        entry.Profile,
			Path:           entry.Path,
			PayloadPath:    entry.PayloadPath,
			SHA256:         entry.SHA256,
			PreviousSHA256: entry.PreviousSHA256,
			Removed:        entry.Removed,
		})
	}
	return out
}

func skillEvidenceForReleaseReport(evidence []skills.SkillProfileSyncEvidence) []UpdateEvidence {
	out := make([]UpdateEvidence, 0, len(evidence))
	for _, ev := range evidence {
		detail := strings.TrimSpace(ev.Profile + "/" + ev.Skill + " " + ev.Reason)
		if ev.Path != "" {
			detail = strings.TrimSpace(detail + " " + ev.Path)
		}
		out = append(out, UpdateEvidence{Kind: UpdateEvidenceReleaseSkillSyncCompleted, Detail: detail})
	}
	return out
}

func formatReleaseSkillSummaries(summaries []skills.SkillProfileSyncSummary) string {
	if len(summaries) == 0 {
		return "no profiles"
	}
	parts := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		parts = append(parts, fmt.Sprintf(
			"%s added=%d updated=%d unchanged=%d conflicts=%d conflict_copies=%d removed=%d orphaned=%d failed=%d",
			summary.Profile,
			summary.Added,
			summary.Updated,
			summary.Unchanged,
			summary.Conflicts,
			summary.ConflictCopies,
			summary.Removed,
			summary.Orphaned,
			summary.Failed,
		))
	}
	return strings.Join(parts, "; ")
}
