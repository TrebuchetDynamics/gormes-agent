package skilltools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestSkillsListToolListsAvailableSkillsAndCategories(t *testing.T) {
	root := t.TempDir()
	writeToolSkillDoc(t, root, "analysis", "careful-review", "Review carefully.")
	writeToolSkillDoc(t, root, "automation", "daily-report", "Prepare a daily report.")
	tool := NewSkillsListTool(SkillsToolsConfig{Root: root})

	result := executeSkillsTool[skillsListToolResult](t, tool, map[string]any{})

	if !result.Success {
		t.Fatalf("success = false, error = %q", result.Error)
	}
	if result.Count != 2 {
		t.Fatalf("count = %d, want 2", result.Count)
	}
	if strings.Join(result.Categories, ",") != "analysis,automation" {
		t.Fatalf("categories = %#v, want analysis, automation", result.Categories)
	}
	if len(result.Skills) != 2 {
		t.Fatalf("len(skills) = %d, want 2: %#v", len(result.Skills), result.Skills)
	}
	got := result.Skills[0]
	if got.Name != "careful-review" || got.Description != "Review carefully." || got.Category != "analysis" || got.Source != "local" {
		t.Fatalf("first skill = %#v, want careful-review local analysis row", got)
	}
}

func TestSkillsListToolFiltersByCategory(t *testing.T) {
	root := t.TempDir()
	writeToolSkillDoc(t, root, "analysis", "careful-review", "Review carefully.")
	writeToolSkillDoc(t, root, "automation", "daily-report", "Prepare a daily report.")
	tool := NewSkillsListTool(SkillsToolsConfig{Root: root})

	result := executeSkillsTool[skillsListToolResult](t, tool, map[string]any{"category": "automation"})

	if !result.Success {
		t.Fatalf("success = false, error = %q", result.Error)
	}
	if result.Count != 1 || len(result.Skills) != 1 || result.Skills[0].Name != "daily-report" {
		t.Fatalf("filtered result = %#v, want only daily-report", result)
	}
	if strings.Join(result.Categories, ",") != "automation" {
		t.Fatalf("categories = %#v, want only automation", result.Categories)
	}
}

func TestSkillViewToolReturnsSkillContentAndLinkedFiles(t *testing.T) {
	root := t.TempDir()
	dir := writeToolSkillDoc(t, root, "analysis", "careful-review", "Review carefully.")
	writeToolSkillFile(t, dir, "references/checklist.md", "Check one.\n")
	writeToolSkillFile(t, dir, "templates/report.yaml", "title: Review\n")
	writeToolSkillFile(t, dir, "scripts/run.sh", "#!/bin/sh\n")
	tool := NewSkillViewTool(SkillsToolsConfig{Root: root})

	result := executeSkillsTool[skillViewToolResult](t, tool, map[string]any{"name": "careful-review"})

	if !result.Success {
		t.Fatalf("success = false, error = %q", result.Error)
	}
	if result.Name != "careful-review" || result.Description != "Review carefully." {
		t.Fatalf("result identity = %#v, want careful-review with description", result)
	}
	if !strings.Contains(result.Content, "Use careful-review.") {
		t.Fatalf("content = %q, want skill body", result.Content)
	}
	if result.LinkedFiles == nil {
		t.Fatal("linked_files is nil, want linked files map")
	}
	lf := *result.LinkedFiles
	if strings.Join(lf["references"], ",") != "references/checklist.md" {
		t.Fatalf("references = %#v, want checklist", lf)
	}
	if strings.Join(lf["templates"], ",") != "templates/report.yaml" {
		t.Fatalf("templates = %#v, want report", lf)
	}
	if strings.Join(lf["scripts"], ",") != "scripts/run.sh" {
		t.Fatalf("scripts = %#v, want run.sh", lf)
	}
	if result.ReadinessStatus != "available" {
		t.Fatalf("readiness_status = %q, want available", result.ReadinessStatus)
	}
}

func TestSkillViewToolReadsLinkedFileAndRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	dir := writeToolSkillDoc(t, root, "analysis", "careful-review", "Review carefully.")
	writeToolSkillFile(t, dir, "references/checklist.md", "Check one.\n")
	secretPath := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("do not leak"), 0o644); err != nil {
		t.Fatalf("WriteFile(secret): %v", err)
	}
	tool := NewSkillViewTool(SkillsToolsConfig{Root: root})

	fileResult := executeSkillsTool[skillViewToolResult](t, tool, map[string]any{
		"name":      "careful-review",
		"file_path": "references/checklist.md",
	})
	if !fileResult.Success || fileResult.File != "references/checklist.md" || fileResult.Content != "Check one.\n" {
		t.Fatalf("file result = %#v, want linked file content", fileResult)
	}

	blocked := executeSkillsTool[skillViewToolResult](t, tool, map[string]any{
		"name":      "careful-review",
		"file_path": "../secret.txt",
	})
	if blocked.Success {
		t.Fatalf("traversal result succeeded: %#v", blocked)
	}
	if strings.Contains(blocked.Error, "do not leak") {
		t.Fatalf("traversal error leaked secret: %q", blocked.Error)
	}

	linkPath := filepath.Join(dir, "references", "secret-link.md")
	if err := os.Symlink(secretPath, linkPath); err == nil {
		symlinkBlocked := executeSkillsTool[skillViewToolResult](t, tool, map[string]any{
			"name":      "careful-review",
			"file_path": "references/secret-link.md",
		})
		if symlinkBlocked.Success {
			t.Fatalf("symlink escape result succeeded: %#v", symlinkBlocked)
		}
		if strings.Contains(symlinkBlocked.Error, "do not leak") || strings.Contains(symlinkBlocked.Content, "do not leak") {
			t.Fatalf("symlink escape leaked secret: %#v", symlinkBlocked)
		}
	} else if !os.IsPermission(err) {
		t.Fatalf("Symlink(%q, %q): %v", secretPath, linkPath, err)
	}
}

func TestSkillsToolsSchemasMatchUpstreamToolNames(t *testing.T) {
	list := NewSkillsListTool(SkillsToolsConfig{})
	if list.Name() != "skills_list" {
		t.Fatalf("list name = %q, want skills_list", list.Name())
	}
	if !json.Valid(list.Schema()) || !strings.Contains(string(list.Schema()), `"category"`) {
		t.Fatalf("list schema = %s, want valid category schema", list.Schema())
	}

	view := NewSkillViewTool(SkillsToolsConfig{})
	if view.Name() != "skill_view" {
		t.Fatalf("view name = %q, want skill_view", view.Name())
	}
	if !json.Valid(view.Schema()) || !strings.Contains(string(view.Schema()), `"file_path"`) {
		t.Fatalf("view schema = %s, want valid file_path schema", view.Schema())
	}
}

func writeToolSkillDoc(t *testing.T, root, category, name, description string) string {
	t.Helper()
	dir := filepath.Join(root, "active", category, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	raw := "---\nname: " + name + "\ndescription: " + description + "\n---\n\nUse " + name + "."
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
	return dir
}

func writeToolSkillFile(t *testing.T, skillDir, rel, content string) {
	t.Helper()
	path := filepath.Join(skillDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func executeSkillsTool[T any](t *testing.T, tool toolkit.Tool, args map[string]any) T {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	out, err := tool.Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result T
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decode output %s: %v", out, err)
	}
	return result
}
