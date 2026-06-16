package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SkillProfileSyncOrphaned = "skill_profile_orphaned"
)

type BundledSkillManifestEntry struct {
	Name           string
	Profile        string
	Path           string
	PayloadPath    string
	SHA256         string
	PreviousSHA256 string
	Removed        bool
}

type BundledSkillManifestSyncRequest struct {
	PayloadRoot string
	Profiles    []SkillProfileRoot
	Entries     []BundledSkillManifestEntry
}

func SyncBundledSkillsFromManifest(ctx context.Context, req BundledSkillManifestSyncRequest) (BundledSkillProfileSyncReport, error) {
	if err := ctx.Err(); err != nil {
		return BundledSkillProfileSyncReport{}, err
	}
	profiles := normalizedSkillProfileRoots(req.Profiles)
	entries := append([]BundledSkillManifestEntry(nil), req.Entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Profile != entries[j].Profile {
			return entries[i].Profile < entries[j].Profile
		}
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Name < entries[j].Name
	})
	report := BundledSkillProfileSyncReport{
		Summaries: make([]SkillProfileSyncSummary, 0, len(profiles)),
	}
	for _, profile := range profiles {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		summary := SkillProfileSyncSummary{Profile: profile.Name}
		if strings.TrimSpace(profile.Root) == "" {
			summary, evidence := invalidProfileRootSyncResult(profile)
			report.Evidence = append(report.Evidence, evidence)
			report.Summaries = append(report.Summaries, summary)
			continue
		}
		for _, entry := range entries {
			if !skillManifestEntryAppliesToProfile(entry, profile.Name) {
				continue
			}
			status := syncOneManifestSkillToProfile(profile, req.PayloadRoot, entry)
			switch status.Reason {
			case "added":
				summary.Added++
			case "updated":
				summary.Updated++
			case "unchanged":
				summary.Unchanged++
			case "removed":
				summary.Removed++
			}
			switch status.Code {
			case SkillProfileSyncConflict:
				summary.Conflicts++
				if status.Path != "" {
					summary.ConflictCopies++
				}
				report.Evidence = append(report.Evidence, status)
			case SkillProfileSyncOrphaned:
				summary.Orphaned++
				report.Evidence = append(report.Evidence, status)
			case SkillProfileSyncWriteFailed, SkillProfileSyncInvalidProfile:
				summary.Failed++
				report.Evidence = append(report.Evidence, status)
			default:
				if status.Code != "" {
					summary.Failed++
					report.Evidence = append(report.Evidence, status)
				}
			}
		}
		report.Summaries = append(report.Summaries, summary)
	}
	sortProfileSyncReport(&report)
	return report, nil
}

func skillManifestEntryAppliesToProfile(entry BundledSkillManifestEntry, profile string) bool {
	want := strings.TrimSpace(entry.Profile)
	return want == "" || want == "*" || want == profile
}

func syncOneManifestSkillToProfile(profile SkillProfileRoot, payloadRoot string, entry BundledSkillManifestEntry) SkillProfileSyncEvidence {
	rel, err := cleanProfileSyncManifestRelPath(entry.Path)
	if err != nil {
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: err.Error()}
	}
	target := filepath.Join(profile.Root, "skills", "active", rel)
	if !profileSyncPathWithin(profile.Root, target) {
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: "target escapes profile root"}
	}
	if entry.Removed {
		return removeManifestSkillFromProfile(profile, entry, target)
	}
	payload, err := readManifestSkillPayload(payloadRoot, entry)
	if err != nil {
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: err.Error()}
	}
	existing, err := os.ReadFile(target)
	switch {
	case err == nil:
		currentSHA := profileSyncSHA256(existing)
		wantSHA := strings.ToLower(strings.TrimSpace(entry.SHA256))
		previousSHA := strings.ToLower(strings.TrimSpace(entry.PreviousSHA256))
		if currentSHA == wantSHA {
			return SkillProfileSyncEvidence{Profile: profile.Name, Skill: entry.Name, Reason: "unchanged"}
		}
		if previousSHA != "" && currentSHA == previousSHA {
			if err := writeProfileSyncFile(target, payload); err != nil {
				return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: err.Error()}
			}
			return SkillProfileSyncEvidence{Profile: profile.Name, Skill: entry.Name, Reason: "updated"}
		}
		conflictPath, err := writeManifestSkillConflictCopy(profile, rel, entry, payload)
		if err != nil {
			return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: err.Error()}
		}
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncConflict,
			Profile: profile.Name,
			Skill:   entry.Name,
			Path:    conflictPath,
			Reason:  "active skill differs from previous bundled digest",
		}
	case !os.IsNotExist(err):
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: "read target failed"}
	}
	if err := writeProfileSyncFile(target, payload); err != nil {
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: err.Error()}
	}
	return SkillProfileSyncEvidence{Profile: profile.Name, Skill: entry.Name, Reason: "added"}
}

func removeManifestSkillFromProfile(profile SkillProfileRoot, entry BundledSkillManifestEntry, target string) SkillProfileSyncEvidence {
	existing, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return SkillProfileSyncEvidence{Profile: profile.Name, Skill: entry.Name, Reason: "unchanged"}
	}
	if err != nil {
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: "read target failed"}
	}
	previousSHA := strings.ToLower(strings.TrimSpace(entry.PreviousSHA256))
	if previousSHA == "" || profileSyncSHA256(existing) != previousSHA {
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncOrphaned,
			Profile: profile.Name,
			Skill:   entry.Name,
			Reason:  "removed bundled skill has user modifications",
		}
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return SkillProfileSyncEvidence{Code: SkillProfileSyncWriteFailed, Profile: profile.Name, Skill: entry.Name, Reason: err.Error()}
	}
	removeEmptyProfileSyncDirs(filepath.Dir(target), filepath.Join(profile.Root, "skills", "active"))
	return SkillProfileSyncEvidence{Profile: profile.Name, Skill: entry.Name, Reason: "removed"}
}

func readManifestSkillPayload(payloadRoot string, entry BundledSkillManifestEntry) ([]byte, error) {
	rel, err := cleanProfileSyncManifestRelPath(entry.PayloadPath)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(payloadRoot, rel)
	if !profileSyncPathWithin(payloadRoot, path) {
		return nil, fmt.Errorf("payload escapes root")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("payload symlink is not allowed")
	}
	if info.IsDir() {
		return nil, fmt.Errorf("payload directory is not allowed")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if got, want := profileSyncSHA256(raw), strings.ToLower(strings.TrimSpace(entry.SHA256)); got != want {
		return nil, fmt.Errorf("payload SHA-256 mismatch: expected %s got %s", want, got)
	}
	return raw, nil
}

func writeManifestSkillConflictCopy(profile SkillProfileRoot, rel string, entry BundledSkillManifestEntry, payload []byte) (string, error) {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filepath.Dir(rel)), string(os.PathSeparator))
	}
	digest := strings.ToLower(strings.TrimSpace(entry.SHA256))
	if len(digest) > 12 {
		digest = digest[:12]
	}
	target := filepath.Join(profile.Root, "skills", ".bundled-conflicts", name, digest, rel)
	if !profileSyncPathWithin(profile.Root, target) {
		return "", fmt.Errorf("conflict target escapes profile root")
	}
	if err := writeProfileSyncFile(target, payload); err != nil {
		return "", err
	}
	return target, nil
}

func writeProfileSyncFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func cleanProfileSyncManifestRelPath(path string) (string, error) {
	path = filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if path == "." || path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe relative path %q", path)
	}
	return path, nil
}

func profileSyncSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func removeEmptyProfileSyncDirs(dir, stop string) {
	for profileSyncPathWithin(stop, dir) && dir != stop {
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
