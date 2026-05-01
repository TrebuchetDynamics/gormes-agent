package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// =============================================================================
// Helpers
// =============================================================================

func executeSkillManage(t *testing.T, tool Tool, args map[string]any) skillManageResult {
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
