package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncBundledSkillsFromManifestUsesDigestThreeWayAndConflictCopies(t *testing.T) {
	payloadRoot := t.TempDir()
	oldBody := profileSyncSkillDoc("reviewer", "Review old docs")
	newBody := profileSyncSkillDoc("reviewer", "Review new docs")
	writeProfileSyncRawFile(t, payloadRoot, "skills/reviewer/SKILL.md", newBody)

	defaultRoot := t.TempDir()
	workRoot := t.TempDir()
	assertWriteProfileSkill(t, defaultRoot, "productivity", "reviewer", oldBody)
	assertWriteProfileSkill(t, workRoot, "productivity", "reviewer", "operator edited reviewer")

	report, err := SyncBundledSkillsFromManifest(context.Background(), BundledSkillManifestSyncRequest{
		PayloadRoot: payloadRoot,
		Profiles: []SkillProfileRoot{
			{Name: "work", Root: workRoot},
			{Name: "default", Root: defaultRoot},
		},
		Entries: []BundledSkillManifestEntry{{
			Name:           "reviewer",
			Path:           "productivity/reviewer/SKILL.md",
			PayloadPath:    "skills/reviewer/SKILL.md",
			SHA256:         releaseSkillTestSHA256(newBody),
			PreviousSHA256: releaseSkillTestSHA256(oldBody),
		}},
	})
	if err != nil {
		t.Fatalf("SyncBundledSkillsFromManifest() error = %v", err)
	}

	assertFileText(t, profileSkillPath(defaultRoot, "productivity", "reviewer"), newBody)
	assertFileText(t, profileSkillPath(workRoot, "productivity", "reviewer"), "operator edited reviewer")
	conflictPath := filepath.Join(workRoot, "skills", ".bundled-conflicts", "reviewer", releaseSkillTestSHA256(newBody)[:12], "productivity", "reviewer", "SKILL.md")
	assertFileText(t, conflictPath, newBody)
	if report.Summaries[0].Profile != "default" || report.Summaries[0].Updated != 1 {
		t.Fatalf("default summary = %+v, want one updated", report.Summaries[0])
	}
	if report.Summaries[1].Profile != "work" || report.Summaries[1].Conflicts != 1 || report.Summaries[1].ConflictCopies != 1 {
		t.Fatalf("work summary = %+v, want one conflict copy", report.Summaries[1])
	}
}

func TestSyncBundledSkillsFromManifestRemovesOnlyUnmodifiedSkills(t *testing.T) {
	oldBody := profileSyncSkillDoc("legacy", "Legacy skill")
	defaultRoot := t.TempDir()
	workRoot := t.TempDir()
	assertWriteProfileSkill(t, defaultRoot, "ops", "legacy", oldBody)
	assertWriteProfileSkill(t, workRoot, "ops", "legacy", "operator kept legacy")

	report, err := SyncBundledSkillsFromManifest(context.Background(), BundledSkillManifestSyncRequest{
		Profiles: []SkillProfileRoot{
			{Name: "default", Root: defaultRoot},
			{Name: "work", Root: workRoot},
		},
		Entries: []BundledSkillManifestEntry{{
			Name:           "legacy",
			Path:           "ops/legacy/SKILL.md",
			PreviousSHA256: releaseSkillTestSHA256(oldBody),
			Removed:        true,
		}},
	})
	if err != nil {
		t.Fatalf("SyncBundledSkillsFromManifest() error = %v", err)
	}

	assertProfileSkillMissing(t, defaultRoot, "ops", "legacy")
	assertFileText(t, profileSkillPath(workRoot, "ops", "legacy"), "operator kept legacy")
	if report.Summaries[0].Profile != "default" || report.Summaries[0].Removed != 1 {
		t.Fatalf("default summary = %+v, want one removed", report.Summaries[0])
	}
	if report.Summaries[1].Profile != "work" || report.Summaries[1].Orphaned != 1 {
		t.Fatalf("work summary = %+v, want one orphaned", report.Summaries[1])
	}
}

func writeProfileSyncRawFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func releaseSkillTestSHA256(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func assertProfileSkillMissing(t *testing.T, root, category, name string) {
	t.Helper()
	path := profileSkillPath(root, category, name)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("profile skill %q exists or stat failed unexpectedly: %v", path, err)
	}
}
