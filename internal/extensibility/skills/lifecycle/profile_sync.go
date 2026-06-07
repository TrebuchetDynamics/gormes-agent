package lifecycle

import (
	"bytes"
	"context"
	"fmt"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SkillProfileSyncConflict       = "skill_profile_conflict"
	SkillProfileSyncWriteFailed    = "skill_profile_write_failed"
	SkillProfileSyncUnavailable    = "skill_profile_sync_unavailable"
	SkillProfileSyncInvalidProfile = "skill_profile_invalid"
)

type SkillProfileRoot struct {
	Name string
	Root string
}

type BundledSkillProfileSyncRequest struct {
	BundledRoot string
	Profiles    []SkillProfileRoot
}

type BundledSkillProfileSyncReport struct {
	Summaries []SkillProfileSyncSummary
	Evidence  []SkillProfileSyncEvidence
}

type SkillProfileSyncSummary struct {
	Profile        string
	Added          int
	Updated        int
	Unchanged      int
	Conflicts      int
	ConflictCopies int
	Removed        int
	Orphaned       int
	Failed         int
}

type SkillProfileSyncEvidence struct {
	Code    string
	Profile string
	Skill   string
	Path    string
	Reason  string
}

type bundledSkillForProfileSync struct {
	Name   string
	RelDir string
	Raw    []byte
}

func SyncBundledSkillsToProfiles(ctx context.Context, req BundledSkillProfileSyncRequest) (BundledSkillProfileSyncReport, error) {
	if err := ctx.Err(); err != nil {
		return BundledSkillProfileSyncReport{}, err
	}
	bundled, err := bundledSkillsForProfileSync(req.BundledRoot)
	if err != nil {
		return BundledSkillProfileSyncReport{}, err
	}
	profiles := normalizedSkillProfileRoots(req.Profiles)

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
		for _, skill := range bundled {
			status := syncOneBundledSkillToProfile(profile, skill)
			switch status.Code {
			case "":
				switch status.Reason {
				case "unchanged":
					summary.Unchanged++
				default:
					summary.Added++
				}
			case SkillProfileSyncConflict:
				summary.Conflicts++
				report.Evidence = append(report.Evidence, status)
			default:
				summary.Failed++
				report.Evidence = append(report.Evidence, status)
			}
		}
		report.Summaries = append(report.Summaries, summary)
	}
	sortProfileSyncReport(&report)
	return report, nil
}

func invalidProfileRootSyncResult(profile SkillProfileRoot) (SkillProfileSyncSummary, SkillProfileSyncEvidence) {
	return SkillProfileSyncSummary{Profile: profile.Name, Failed: 1}, SkillProfileSyncEvidence{
		Code:    SkillProfileSyncInvalidProfile,
		Profile: profile.Name,
		Reason:  "profile root is empty",
	}
}

func bundledSkillsForProfileSync(root string) ([]bundledSkillForProfileSync, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	var paths []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "SKILL.md" {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Strings(paths)

	out := make([]bundledSkillForProfileSync, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		skill, err := document.Parse(raw, document.DefaultMaxDocumentBytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		out = append(out, bundledSkillForProfileSync{
			Name:   skill.Name,
			RelDir: filepath.Clean(rel),
			Raw:    raw,
		})
	}
	return out, nil
}

func normalizedSkillProfileRoots(in []SkillProfileRoot) []SkillProfileRoot {
	out := make([]SkillProfileRoot, 0, len(in))
	seen := map[string]bool{}
	for _, profile := range in {
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Root = strings.TrimSpace(profile.Root)
		if profile.Name == "" {
			profile.Name = "main"
		}
		key := profile.Name + "\x00" + profile.Root
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, profile)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Root < out[j].Root
	})
	return out
}

func syncOneBundledSkillToProfile(profile SkillProfileRoot, skill bundledSkillForProfileSync) SkillProfileSyncEvidence {
	target := filepath.Join(profile.Root, "skills", "active", skill.RelDir, "SKILL.md")
	if !profileSyncPathWithin(profile.Root, target) {
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncWriteFailed,
			Profile: profile.Name,
			Skill:   skill.Name,
			Reason:  "target escapes profile root",
		}
	}
	existing, err := os.ReadFile(target)
	switch {
	case err == nil:
		if bytes.Equal(existing, skill.Raw) {
			return SkillProfileSyncEvidence{Profile: profile.Name, Skill: skill.Name, Reason: "unchanged"}
		}
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncConflict,
			Profile: profile.Name,
			Skill:   skill.Name,
			Reason:  "existing skill differs from bundled skill",
		}
	case !os.IsNotExist(err):
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncWriteFailed,
			Profile: profile.Name,
			Skill:   skill.Name,
			Reason:  "read target failed",
		}
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncWriteFailed,
			Profile: profile.Name,
			Skill:   skill.Name,
			Reason:  "create target directory failed",
		}
	}
	if err := os.WriteFile(target, skill.Raw, 0o644); err != nil {
		return SkillProfileSyncEvidence{
			Code:    SkillProfileSyncWriteFailed,
			Profile: profile.Name,
			Skill:   skill.Name,
			Reason:  "write target failed",
		}
	}
	return SkillProfileSyncEvidence{Profile: profile.Name, Skill: skill.Name, Reason: "added"}
}

func sortProfileSyncReport(report *BundledSkillProfileSyncReport) {
	sort.SliceStable(report.Summaries, func(i, j int) bool {
		return report.Summaries[i].Profile < report.Summaries[j].Profile
	})
	sort.SliceStable(report.Evidence, func(i, j int) bool {
		if report.Evidence[i].Profile != report.Evidence[j].Profile {
			return report.Evidence[i].Profile < report.Evidence[j].Profile
		}
		if report.Evidence[i].Skill != report.Evidence[j].Skill {
			return report.Evidence[i].Skill < report.Evidence[j].Skill
		}
		return report.Evidence[i].Code < report.Evidence[j].Code
	})
}

func profileSyncPathWithin(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
