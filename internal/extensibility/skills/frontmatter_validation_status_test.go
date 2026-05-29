package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeBuildSkillBlock_InvalidFrontmatterExcludedAndStatusReported(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "active", "review-tests", "SKILL.md"), "review-tests", "Review tests carefully", "Check assertions before implementation.")

	invalidDir := filepath.Join(root, "active", "broken-skill")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	invalidPath := filepath.Join(invalidDir, "SKILL.md")
	invalidRaw := strings.Join([]string{
		"---",
		"name: broken-skill",
		"description: missing close",
		"# stray heading inside frontmatter",
		"",
		"body",
	}, "\n")
	if err := os.WriteFile(invalidPath, []byte(invalidRaw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	runtime := NewRuntime(root, 8*1024, 5, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "review tests", RuntimeOptions{})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions error = %v", err)
	}

	if strings.Contains(block, "broken-skill") {
		t.Fatalf("rendered block leaks invalid skill: %q", block)
	}
	if len(names) != 1 || names[0] != "review-tests" {
		t.Fatalf("names = %#v, want only valid skill", names)
	}

	var invalidStatus *SkillStatus
	for i := range statuses {
		if statuses[i].Status == SkillStatusFrontmatterInvalid {
			invalidStatus = &statuses[i]
			break
		}
	}
	if invalidStatus == nil {
		t.Fatalf("statuses = %+v, want one with SkillStatusFrontmatterInvalid", statuses)
	}
	if invalidStatus.Path != invalidPath {
		t.Fatalf("invalid status Path = %q, want %q", invalidStatus.Path, invalidPath)
	}
	if !strings.Contains(invalidStatus.Reason, "MISSING_CLOSE") {
		t.Fatalf("invalid status Reason = %q, want it to include MISSING_CLOSE evidence", invalidStatus.Reason)
	}
}

func TestSkillStoreSnapshot_PreservesValidSkillsAroundInvalidSibling(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "active", "alpha", "SKILL.md"), "alpha", "Alpha skill", "Body A.")
	writeSkillDoc(t, filepath.Join(root, "active", "bravo", "SKILL.md"), "bravo", "Bravo skill", "Body B.")

	brokenDir := filepath.Join(root, "active", "no-frontmatter")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "SKILL.md"), []byte("# only a heading\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewStore(root, 8*1024)
	snap, err := store.SnapshotActive()
	if err != nil {
		t.Fatalf("SnapshotActive error = %v", err)
	}
	if len(snap.Skills) != 2 {
		t.Fatalf("snap.Skills len = %d, want 2 valid skills (invalid sibling skipped)", len(snap.Skills))
	}
	if len(snap.Invalid) != 1 {
		t.Fatalf("snap.Invalid len = %d, want 1 entry for the malformed file", len(snap.Invalid))
	}
	if snap.Invalid[0].Path != filepath.Join(brokenDir, "SKILL.md") {
		t.Fatalf("Invalid[0].Path = %q, want broken sibling", snap.Invalid[0].Path)
	}
	if codeOf(snap.Invalid[0].Errors, SkillValidationMissingOpen) == nil {
		t.Fatalf("Invalid[0].Errors = %+v, want MISSING_OPEN classification", snap.Invalid[0].Errors)
	}
}
