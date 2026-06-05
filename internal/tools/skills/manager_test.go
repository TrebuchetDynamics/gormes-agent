package skilltools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillspkg "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestSkillCreateWritesValidSKILLMD(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	content := `---
name: my-test-skill
description: A test skill for unit testing.
---

# My Test Skill

This skill does something useful.

## Steps

1. First step
2. Second step`

	result := executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "my-test-skill",
		"content": content,
	})

	if !result.Success {
		t.Fatalf("create failed: %s", result.Error)
	}

	// Verify file was created
	skillPath := filepath.Join(root, "active", "my-test-skill", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	want := ensurePortableFrontmatter(content, "my-test-skill")
	if string(data) != want {
		t.Fatalf("SKILL.md content mismatch.\ngot: %q\nwant: %q", string(data), want)
	}
}

func TestSkillCreateWithCategory(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	content := `---
name: categorized-skill
description: A skill in a category.
---

# Categorized Skill`

	result := executeSkillManage(t, tool, map[string]any{
		"action":   "create",
		"name":     "categorized-skill",
		"content":  content,
		"category": "testing",
	})

	if !result.Success {
		t.Fatalf("create with category failed: %s", result.Error)
	}

	// Verify file was created in category
	skillPath := filepath.Join(root, "active", "testing", "categorized-skill", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatalf("SKILL.md not found in category: %v", err)
	}
}

func TestSkillEditUpdatesContent(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	// Create initial skill
	initialContent := `---
name: edit-test
description: Initial description.
---

# Initial Content`

	executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "edit-test",
		"content": initialContent,
	})

	// Edit the skill
	newContent := `---
name: edit-test
description: Updated description.
---

# Updated Content

This is the new body.`

	result := executeSkillManage(t, tool, map[string]any{
		"action":  "edit",
		"name":    "edit-test",
		"content": newContent,
	})

	if !result.Success {
		t.Fatalf("edit failed: %s", result.Error)
	}

	// Verify new content was written
	skillPath := filepath.Join(root, "active", "edit-test", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	if string(data) != newContent {
		t.Fatalf("SKILL.md content not updated.\ngot: %q\nwant: %q", string(data), newContent)
	}
}

func TestSkillPatchAddsNewSection(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	// Create initial skill
	initialContent := `---
name: patch-test
description: A skill to patch.
---

# Original Content

## Steps

1. Step one`

	executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "patch-test",
		"content": initialContent,
	})

	// Patch to add a new section
	patchResult := executeSkillManage(t, tool, map[string]any{
		"action":     "patch",
		"name":       "patch-test",
		"old_string": "## Steps\n\n1. Step one",
		"new_string": "## Steps\n\n1. Step one\n2. Step two",
	})

	if !patchResult.Success {
		t.Fatalf("patch failed: %s", patchResult.Error)
	}

	// Verify patch was applied
	skillPath := filepath.Join(root, "active", "patch-test", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}

	if !strings.Contains(string(data), "Step two") {
		t.Fatalf("patch not applied, content: %q", string(data))
	}
}

func TestSkillDeleteRemovesFile(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	// Create a skill
	content := `---
name: delete-test
description: A skill to delete.
---

# Delete Me`

	executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "delete-test",
		"content": content,
	})

	// Verify it exists
	skillPath := filepath.Join(root, "active", "delete-test", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatal("skill was not created")
	}

	// Delete it
	result := executeSkillManage(t, tool, map[string]any{
		"action": "delete",
		"name":   "delete-test",
	})

	if !result.Success {
		t.Fatalf("delete failed: %s", result.Error)
	}

	// Verify it's gone
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("skill was not deleted")
	}
}

func TestSkillCreateRejectsDuplicateName(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	content := `---
name: duplicate-test
description: First one.
---

# First`

	executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "duplicate-test",
		"content": content,
	})

	// Try to create another with same name
	result := executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "duplicate-test",
		"content": content,
	})

	if result.Success {
		t.Fatal("expected duplicate name to be rejected")
	}
	if !strings.Contains(result.Error, "already exists") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestSkillCreateRejectsInvalidName(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	tests := []struct {
		name    string
		wantErr string
	}{
		{"", "required"},
		{"UPPER", "lowercase"},
		{"has space", "lowercase"},
		{"has/slash", "lowercase"},
		{"-starts-with-hyphen", "start with a letter or digit"},
		{".starts-with-dot", "start with a letter or digit"},
		{strings.Repeat("a", 65), "exceeds 64 characters"},
	}

	content := `---
name: test
description: Test.
---

# Test`

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := executeSkillManage(t, tool, map[string]any{
				"action":  "create",
				"name":    tc.name,
				"content": content,
			})
			if result.Success {
				t.Errorf("expected failure for name %q", tc.name)
			}
			if !strings.Contains(result.Error, tc.wantErr) {
				t.Errorf("error %q does not contain %q", result.Error, tc.wantErr)
			}
		})
	}
}

func TestSkillCreateRejectsEmptyContent(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	result := executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    "empty-test",
		"content": "",
	})

	if result.Success {
		t.Fatal("expected empty content to be rejected")
	}
	if !strings.Contains(result.Error, "required") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestSkillCreateRejectsMissingFrontmatter(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	tests := []struct {
		name    string
		content string
	}{
		{"no frontmatter", "# Just a header\n\nContent"},
		{"no closing delimiter", "---\nname: test\ndescription: Test.\n\n# Content"},
		{"no name field", "---\ndescription: Test.\n---\n\n# Content"},
		{"no description", "---\nname: test\n---\n\n# Content"},
		{"empty body", "---\nname: test\ndescription: Test.\n---\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := executeSkillManage(t, tool, map[string]any{
				"action":  "create",
				"name":    "test-skill",
				"content": tc.content,
			})
			if result.Success {
				t.Errorf("expected failure for %s", tc.name)
			}
		})
	}
}

func TestSkillManageUnknownAction(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	result := executeSkillManage(t, tool, map[string]any{
		"action": "unknown",
		"name":   "test",
	})

	if result.Success {
		t.Fatal("expected unknown action to be rejected")
	}
	if !strings.Contains(result.Error, "Unknown action") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestSkillEditNotFound(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	result := executeSkillManage(t, tool, map[string]any{
		"action":  "edit",
		"name":    "nonexistent",
		"content": "---\nname: test\ndescription: Test.\n---\n\n# Test",
	})

	if result.Success {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(result.Error, "not found") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestSkillPatchNotFound(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	result := executeSkillManage(t, tool, map[string]any{
		"action":     "patch",
		"name":       "nonexistent",
		"old_string": "something",
		"new_string": "replacement",
	})

	if result.Success {
		t.Fatal("expected not found error")
	}
}

func TestSkillDeleteNotFound(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	result := executeSkillManage(t, tool, map[string]any{
		"action": "delete",
		"name":   "nonexistent",
	})

	if result.Success {
		t.Fatal("expected not found error")
	}
}

func TestSkillPatchOldStringNotFound(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})

	// Create a skill
	executeSkillManage(t, tool, map[string]any{
		"action": "create",
		"name":   "patch-notfound",
		"content": `---
name: patch-notfound
description: Test.
---

# Content`,
	})

	result := executeSkillManage(t, tool, map[string]any{
		"action":     "patch",
		"name":       "patch-notfound",
		"old_string": "nonexistent string that is not in the file",
		"new_string": "replacement",
	})

	if result.Success {
		t.Fatal("expected old_string not found error")
	}
	if !strings.Contains(result.Error, "not found") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestSkillManagerSchema_HermesActions(t *testing.T) {
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: t.TempDir()})
	schema := string(tool.Schema())
	for _, want := range []string{
		`"create"`,
		`"patch"`,
		`"edit"`,
		`"delete"`,
		`"write_file"`,
		`"remove_file"`,
		`"file_path"`,
		`"file_content"`,
		`"replace_all"`,
		`"category"`,
		`"absorbed_into"`,
	} {
		if !strings.Contains(schema, want) {
			t.Fatalf("schema missing %s:\n%s", want, schema)
		}
	}
}

func TestSkillManagerSupportFile_WritePatchRemove(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})
	createSkillForManagerTest(t, tool, "support-skill")

	written := executeSkillManage(t, tool, map[string]any{
		"action":       "write_file",
		"name":         "support-skill",
		"file_path":    "references/api.md",
		"file_content": "old endpoint text",
	})
	if !written.Success {
		t.Fatalf("write_file failed: %s", written.Error)
	}
	supportPath := filepath.Join(root, "active", "support-skill", "references", "api.md")
	if got := readTextForManagerTest(t, supportPath); got != "old endpoint text" {
		t.Fatalf("support file = %q", got)
	}

	patched := executeSkillManage(t, tool, map[string]any{
		"action":      "patch",
		"name":        "support-skill",
		"file_path":   "references/api.md",
		"old_string":  "old endpoint",
		"new_string":  "new endpoint",
		"replace_all": false,
	})
	if !patched.Success {
		t.Fatalf("support-file patch failed: %s", patched.Error)
	}
	if got := readTextForManagerTest(t, supportPath); got != "new endpoint text" {
		t.Fatalf("patched support file = %q", got)
	}

	removed := executeSkillManage(t, tool, map[string]any{
		"action":    "remove_file",
		"name":      "support-skill",
		"file_path": "references/api.md",
	})
	if !removed.Success {
		t.Fatalf("remove_file failed: %s", removed.Error)
	}
	if _, err := os.Stat(supportPath); !os.IsNotExist(err) {
		t.Fatalf("support file still exists after remove_file, stat err=%v", err)
	}

	available := executeSkillManage(t, tool, map[string]any{
		"action":       "write_file",
		"name":         "support-skill",
		"file_path":    "templates/available.txt",
		"file_content": "available",
	})
	if !available.Success {
		t.Fatalf("write available file failed: %s", available.Error)
	}
	missing := executeSkillManage(t, tool, map[string]any{
		"action":    "remove_file",
		"name":      "support-skill",
		"file_path": "references/missing.md",
	})
	if missing.Success || !strings.Contains(missing.Error, "not found") {
		t.Fatalf("missing remove = %+v, want not found", missing)
	}
	if !stringSliceContains(missing.AvailableFiles, "templates/available.txt") {
		t.Fatalf("available files = %#v, want templates/available.txt", missing.AvailableFiles)
	}
}

func TestSkillManagerSupportFile_PathSafety(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})
	createSkillForManagerTest(t, tool, "safe-skill")

	for _, filePath := range []string{
		"../escape.md",
		"references/../../../escape.md",
		filepath.Join(root, "active", "safe-skill", "references", "abs.md"),
		"secret/hidden.md",
		"references",
		"malicious.md",
	} {
		t.Run(filePath, func(t *testing.T) {
			result := executeSkillManage(t, tool, map[string]any{
				"action":       "write_file",
				"name":         "safe-skill",
				"file_path":    filePath,
				"file_content": "nope",
			})
			if result.Success {
				t.Fatalf("write_file(%q) succeeded, want safety refusal", filePath)
			}
		})
	}

	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	linkParent := filepath.Join(root, "active", "safe-skill", "references")
	if err := os.MkdirAll(linkParent, 0o755); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(linkParent, "escape")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	result := executeSkillManage(t, tool, map[string]any{
		"action":       "write_file",
		"name":         "safe-skill",
		"file_path":    "references/escape/owned.md",
		"file_content": "nope",
	})
	if result.Success || !strings.Contains(strings.ToLower(result.Error), "escape") {
		t.Fatalf("symlink escape result = %+v, want escape refusal", result)
	}
	if _, err := os.Stat(filepath.Join(outside, "owned.md")); !os.IsNotExist(err) {
		t.Fatalf("outside file was written, stat err=%v", err)
	}
}

func TestSkillManagerAbsorbedIntoDelete(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})
	createSkillForManagerTest(t, tool, "narrow")
	createSkillForManagerTest(t, tool, "umbrella")

	self := executeSkillManage(t, tool, map[string]any{
		"action":        "delete",
		"name":          "narrow",
		"absorbed_into": "narrow",
	})
	if self.Success || !strings.Contains(self.Error, "cannot equal") {
		t.Fatalf("self absorbed_into = %+v, want refusal", self)
	}
	ghost := executeSkillManage(t, tool, map[string]any{
		"action":        "delete",
		"name":          "narrow",
		"absorbed_into": "ghost",
	})
	if ghost.Success || !strings.Contains(ghost.Error, "does not exist") {
		t.Fatalf("ghost absorbed_into = %+v, want refusal", ghost)
	}
	deleted := executeSkillManage(t, tool, map[string]any{
		"action":        "delete",
		"name":          "narrow",
		"absorbed_into": "umbrella",
	})
	if !deleted.Success || !strings.Contains(deleted.Message, "absorbed into \"umbrella\"") {
		t.Fatalf("delete absorbed_into = %+v, want success with target", deleted)
	}

	createSkillForManagerTest(t, tool, "pruned")
	pruned := executeSkillManage(t, tool, map[string]any{
		"action":        "delete",
		"name":          "pruned",
		"absorbed_into": "   ",
	})
	if !pruned.Success || strings.Contains(pruned.Message, "absorbed into") {
		t.Fatalf("prune delete = %+v, want success without absorbed target", pruned)
	}
}

func TestSkillManagerPinnedGuard(t *testing.T) {
	root := t.TempDir()
	tool := NewSkillManagerTool(SkillManagerToolConfig{Root: root})
	createSkillForManagerTest(t, tool, "pinned-one")
	createSkillForManagerTest(t, tool, "editable-one")
	if err := skillspkg.SetPinned(root, "pinned-one", true); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}

	for _, args := range []map[string]any{
		{"action": "edit", "name": "pinned-one", "content": validSkillContentForManagerTest("pinned-one", "new")},
		{"action": "patch", "name": "pinned-one", "old_string": "# pinned-one", "new_string": "# updated"},
		{"action": "write_file", "name": "pinned-one", "file_path": "references/api.md", "file_content": "x"},
		{"action": "remove_file", "name": "pinned-one", "file_path": "references/api.md"},
		{"action": "delete", "name": "pinned-one"},
	} {
		result := executeSkillManage(t, tool, args)
		if result.Success || !strings.Contains(result.Error, "pinned") || !strings.Contains(result.Hint, "gormes curator unpin pinned-one") {
			t.Fatalf("pinned action %#v = %+v, want pinned refusal with hint", args, result)
		}
	}

	editable := executeSkillManage(t, tool, map[string]any{
		"action":     "patch",
		"name":       "editable-one",
		"old_string": "# editable-one",
		"new_string": "# changed",
	})
	if !editable.Success {
		t.Fatalf("unpinned sibling patch failed: %s", editable.Error)
	}
}

func TestSkillManagerAgentCreatedProvenance(t *testing.T) {
	root := t.TempDir()
	foreground := NewSkillManagerTool(SkillManagerToolConfig{Root: root})
	createSkillForManagerTest(t, foreground, "user-skill")
	if skillspkg.IsAgentCreated(root, "user-skill") {
		t.Fatal("foreground create marked user-skill as agent-created")
	}

	background := NewSkillManagerTool(SkillManagerToolConfig{Root: root, WriteOrigin: SkillWriteOriginBackgroundReview})
	createSkillForManagerTest(t, background, "agent-skill")
	if !skillspkg.IsAgentCreated(root, "agent-skill") {
		t.Fatal("background-review create did not mark agent-skill as agent-created")
	}

	for _, args := range []map[string]any{
		{"action": "edit", "name": "agent-skill", "content": validSkillContentForManagerTest("agent-skill", "edited")},
		{"action": "write_file", "name": "agent-skill", "file_path": "references/api.md", "file_content": "old"},
		{"action": "patch", "name": "agent-skill", "file_path": "references/api.md", "old_string": "old", "new_string": "new"},
		{"action": "remove_file", "name": "agent-skill", "file_path": "references/api.md"},
	} {
		if result := executeSkillManage(t, background, args); !result.Success {
			t.Fatalf("mutation %#v failed: %+v", args, result)
		}
	}
	record, err := skillspkg.GetUsageRecord(root, "agent-skill")
	if err != nil {
		t.Fatalf("GetUsageRecord: %v", err)
	}
	if record.PatchCount != 4 {
		t.Fatalf("PatchCount = %d, want 4", record.PatchCount)
	}

	if result := executeSkillManage(t, background, map[string]any{"action": "delete", "name": "agent-skill"}); !result.Success {
		t.Fatalf("delete agent-skill: %+v", result)
	}
	if _, err := skillspkg.GetUsageRecord(root, "agent-skill"); !errors.Is(err, skillspkg.ErrUsageRecordNotFound) {
		t.Fatalf("usage record after delete err = %v, want ErrUsageRecordNotFound", err)
	}
}

func TestSkillManagerAgentCreatedGuard(t *testing.T) {
	root := t.TempDir()
	calls := 0
	scanner := func(string) error {
		calls++
		return errors.New("blocked by guard")
	}
	unguarded := NewSkillManagerTool(SkillManagerToolConfig{Root: root, GuardScanner: scanner})
	createSkillForManagerTest(t, unguarded, "unguarded")
	if calls != 0 {
		t.Fatalf("guard scanner called while GuardAgentCreated=false")
	}

	guarded := NewSkillManagerTool(SkillManagerToolConfig{
		Root:              root,
		GuardAgentCreated: true,
		GuardScanner:      scanner,
	})
	result := executeSkillManage(t, guarded, map[string]any{
		"action":  "create",
		"name":    "blocked-create",
		"content": validSkillContentForManagerTest("blocked-create", "blocked"),
	})
	if result.Success || !strings.Contains(result.Error, "blocked by guard") {
		t.Fatalf("blocked create = %+v, want guard error", result)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "blocked-create", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("blocked skill left on disk, stat err=%v", err)
	}

	createSkillForManagerTest(t, unguarded, "rollback-skill")
	original := readTextForManagerTest(t, filepath.Join(root, "active", "rollback-skill", "SKILL.md"))
	edit := executeSkillManage(t, guarded, map[string]any{
		"action":  "edit",
		"name":    "rollback-skill",
		"content": validSkillContentForManagerTest("rollback-skill", "blocked edit"),
	})
	if edit.Success || !strings.Contains(edit.Error, "blocked by guard") {
		t.Fatalf("blocked edit = %+v, want guard error", edit)
	}
	if got := readTextForManagerTest(t, filepath.Join(root, "active", "rollback-skill", "SKILL.md")); got != original {
		t.Fatalf("blocked edit did not roll back\n got: %s\nwant: %s", got, original)
	}

	supportPath := filepath.Join(root, "active", "rollback-skill", "references", "guard.md")
	writeOriginal := executeSkillManage(t, unguarded, map[string]any{
		"action":       "write_file",
		"name":         "rollback-skill",
		"file_path":    "references/guard.md",
		"file_content": "original support",
	})
	if !writeOriginal.Success {
		t.Fatalf("write original support file: %+v", writeOriginal)
	}
	newFile := executeSkillManage(t, guarded, map[string]any{
		"action":       "write_file",
		"name":         "rollback-skill",
		"file_path":    "references/new.md",
		"file_content": "blocked new support",
	})
	if newFile.Success || !strings.Contains(newFile.Error, "blocked by guard") {
		t.Fatalf("blocked new support write = %+v, want guard error", newFile)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "rollback-skill", "references", "new.md")); !os.IsNotExist(err) {
		t.Fatalf("blocked new support file left on disk, stat err=%v", err)
	}
	overwrite := executeSkillManage(t, guarded, map[string]any{
		"action":       "write_file",
		"name":         "rollback-skill",
		"file_path":    "references/guard.md",
		"file_content": "blocked overwrite",
	})
	if overwrite.Success || !strings.Contains(overwrite.Error, "blocked by guard") {
		t.Fatalf("blocked support overwrite = %+v, want guard error", overwrite)
	}
	if got := readTextForManagerTest(t, supportPath); got != "original support" {
		t.Fatalf("blocked support overwrite did not roll back: %q", got)
	}
	supportPatch := executeSkillManage(t, guarded, map[string]any{
		"action":     "patch",
		"name":       "rollback-skill",
		"file_path":  "references/guard.md",
		"old_string": "original",
		"new_string": "blocked",
	})
	if supportPatch.Success || !strings.Contains(supportPatch.Error, "blocked by guard") {
		t.Fatalf("blocked support patch = %+v, want guard error", supportPatch)
	}
	if got := readTextForManagerTest(t, supportPath); got != "original support" {
		t.Fatalf("blocked support patch did not roll back: %q", got)
	}
	remove := executeSkillManage(t, guarded, map[string]any{
		"action":    "remove_file",
		"name":      "rollback-skill",
		"file_path": "references/guard.md",
	})
	if remove.Success || !strings.Contains(remove.Error, "blocked by guard") {
		t.Fatalf("blocked support remove = %+v, want guard error", remove)
	}
	if got := readTextForManagerTest(t, supportPath); got != "original support" {
		t.Fatalf("blocked support remove did not restore file: %q", got)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func executeSkillManage(t *testing.T, tool toolkit.Tool, args map[string]any) skillManageResult {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	out, err := tool.Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result skillManageResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	return result
}

func createSkillForManagerTest(t *testing.T, tool toolkit.Tool, name string) {
	t.Helper()
	result := executeSkillManage(t, tool, map[string]any{
		"action":  "create",
		"name":    name,
		"content": validSkillContentForManagerTest(name, name),
	})
	if !result.Success {
		t.Fatalf("create %s: %s", name, result.Error)
	}
}

func validSkillContentForManagerTest(name, heading string) string {
	return `---
name: ` + name + `
description: Test skill ` + name + `.
---

# ` + heading + `

Instructions.
`
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func readTextForManagerTest(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(raw)
}
