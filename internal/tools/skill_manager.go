package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

const (
	SkillManagerToolName       = "skill_manage"
	skillManagerDefaultTimeout = 30 * time.Second

	// Portable SKILL.md frontmatter fields for cross-agent skill sharing.
	// Matching Hermes' skill_manager_tool.py contract with standard metadata.
	skillFrontmatterRequired = `name|description`
	skillFrontmatterOptional = `version|author|date|when|category|tags|requires`

	// skillTemplate is the canonical SKILL.md template auto-populated on create
	// when the agent omits optional fields.
	skillTemplate = `---
name: %s
description: %s
author: gormes-agent
date: %s
version: 1.0.0
---

## Purpose

## Instructions

## Examples

`
)

// SkillManagerToolConfig configures the skill management tool surface.
type SkillManagerToolConfig struct {
	Root    string
	Timeout time.Duration
}

// NewSkillManagerTool returns a skill management tool.
func NewSkillManagerTool(cfg SkillManagerToolConfig) Tool {
	return &SkillManagerTool{cfg: normalizeSkillManagerConfig(cfg)}
}

type SkillManagerTool struct {
	cfg SkillManagerToolConfig
}

type skillManageArgs struct {
	Action     string `json:"action"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Category   string `json:"category"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
}

type skillManageResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

const (
	maxSkillNameLength   = 64
	maxDescriptionLength = 1024
	maxContentChars      = 100_000
	validNameRE          = `^[a-z0-9][a-z0-9._-]*$`
)

var nameRegex = regexp.MustCompile(validNameRE)

// =============================================================================
// Tool interface
// =============================================================================

func (*SkillManagerTool) Name() string { return SkillManagerToolName }

func (*SkillManagerTool) Description() string {
	return "Manage skills (create, update, delete). Skills are your procedural memory — reusable approaches for recurring task types. New skills go to the skills root; existing skills can be modified or deleted. Actions: create (new skill with full SKILL.md), edit (replace SKILL.md content), patch (find-and-replace within SKILL.md), delete (remove skill)."
}

func (*SkillManagerTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "edit", "patch", "delete"],
				"description": "The action to perform: create, edit, patch, or delete."
			},
			"name": {
				"type": "string",
				"description": "Skill name (lowercase, numbers, hyphens, dots, underscores; 64 chars max; must start with letter or digit)."
			},
			"content": {
				"type": "string",
				"description": "Full SKILL.md content (YAML frontmatter + markdown body). Required for create and edit."
			},
			"category": {
				"type": "string",
				"description": "Optional category for organizing the skill (e.g., 'devops', 'data-science'). Only used with create."
			},
			"old_string": {
				"type": "string",
				"description": "Text to find in the file (required for patch). Must be unique unless replace_all=true."
			},
			"new_string": {
				"type": "string",
				"description": "Replacement text for patch. Can be empty to delete matched text."
			},
			"replace_all": {
				"type": "boolean",
				"description": "For patch: replace all occurrences instead of requiring unique match (default: false)."
			}
		},
		"required": ["action", "name"]
	}`)
}

func (t *SkillManagerTool) Timeout() time.Duration {
	if t == nil || t.cfg.Timeout <= 0 {
		return skillManagerDefaultTimeout
	}
	return t.cfg.Timeout
}

func (t *SkillManagerTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	if len(strings.TrimSpace(string(args))) == 0 {
		return json.Marshal(skillManageResult{Success: false, Error: "arguments required"})
	}
	var in skillManageArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return json.Marshal(skillManageResult{Success: false, Error: "invalid args: " + err.Error()})
	}

	switch strings.ToLower(strings.TrimSpace(in.Action)) {
	case "create":
		return json.Marshal(t.handleCreate(in))
	case "edit":
		return json.Marshal(t.handleEdit(in))
	case "patch":
		return json.Marshal(t.handlePatch(in))
	case "delete":
		return json.Marshal(t.handleDelete(in))
	default:
		return json.Marshal(skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown action %q. Use: create, edit, patch, delete", in.Action),
		})
	}
}

// =============================================================================
// Config normalization
// =============================================================================

func normalizeSkillManagerConfig(cfg SkillManagerToolConfig) SkillManagerToolConfig {
	if cfg.Root == "" {
		cfg.Root = skills.DefaultRoot()
	}
	if cfg.Root != "" {
		// Ensure active dir exists
		activeDir := filepath.Join(cfg.Root, "active")
		_ = os.MkdirAll(activeDir, 0o755)
	}
	return cfg
}

// =============================================================================
// Validation helpers
// =============================================================================

func validateSkillName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Skill name is required."
	}
	if len(name) > maxSkillNameLength {
		return fmt.Sprintf("Skill name exceeds %d characters.", maxSkillNameLength)
	}
	if !nameRegex.MatchString(name) {
		return "Invalid skill name. Use lowercase letters, numbers, hyphens, dots, and underscores. Must start with a letter or digit."
	}
	return ""
}

func validateCategory(category string) string {
	if category == "" {
		return ""
	}
	category = strings.TrimSpace(category)
	if category == "" {
		return ""
	}
	if strings.Contains(category, "/") || strings.Contains(category, "\\") {
		return "Category must be a single directory name, not a path."
	}
	if len(category) > maxSkillNameLength {
		return fmt.Sprintf("Category exceeds %d characters.", maxSkillNameLength)
	}
	if !nameRegex.MatchString(category) {
		return "Invalid category name. Use lowercase letters, numbers, hyphens, dots, and underscores."
	}
	return ""
}

func validateSkillContent(content string) string {
	if content == "" {
		return "Content is required."
	}
	if len(content) > maxContentChars {
		return fmt.Sprintf("Content exceeds %d characters (limit: %d). Consider splitting into smaller skill with supporting files.", len(content), maxContentChars)
	}
	// Check frontmatter structure
	if !strings.HasPrefix(content, "---") {
		return "SKILL.md must start with YAML frontmatter (---)."
	}
	idx := strings.Index(content[3:], "\n---")
	if idx < 0 {
		return "SKILL.md frontmatter is not closed. Ensure you have a closing '---' line."
	}
	frontmatterEnd := 3 + idx + len("\n---")
	frontmatter := content[3 : 3+idx]
	if strings.TrimSpace(frontmatter) == "" {
		return "Frontmatter block is empty."
	}
	// Validate required frontmatter fields
	frontmatterLines := strings.Split(frontmatter, "\n")
	hasName := false
	hasDescription := false
	for _, line := range frontmatterLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "name:") {
			hasName = true
		}
		if strings.HasPrefix(trimmed, "description:") {
			hasDescription = true
		}
	}
	if !hasName {
		return "Frontmatter must include 'name' field."
	}
	if !hasDescription {
		return "Frontmatter must include 'description' field."
	}
	// Check body exists after frontmatter
	body := strings.TrimSpace(content[frontmatterEnd:])
	if body == "" {
		return "SKILL.md must have content after the frontmatter (instructions, procedures, etc.)."
	}
	return ""
}

// =============================================================================
// Action handlers
// =============================================================================

func (t *SkillManagerTool) handleCreate(in skillManageArgs) skillManageResult {
	name := strings.TrimSpace(in.Name)
	if err := validateSkillName(name); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	category := strings.TrimSpace(in.Category)
	if err := validateCategory(category); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	content := strings.TrimSpace(in.Content)
	if err := validateSkillContent(content); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	// Auto-populate missing portable SKILL.md frontmatter fields
	content = ensurePortableFrontmatter(content, name)

	// Check for name collision
	if existing := findSkill(t.cfg.Root, name); existing != "" {
		return skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("A skill named %q already exists at %s.", name, existing),
		}
	}

	// Create skill directory
	skillDir := resolveSkillDir(t.cfg.Root, name, category)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return skillManageResult{Success: false, Error: "failed to create skill directory: " + err.Error()}
	}

	// Write SKILL.md
	skillMD := filepath.Join(skillDir, "SKILL.md")
	if err := atomicWriteText(skillMD, content); err != nil {
		// Clean up directory on failure
		os.RemoveAll(skillDir)
		return skillManageResult{Success: false, Error: "failed to write SKILL.md: " + err.Error()}
	}

	relPath, _ := filepath.Rel(t.cfg.Root, skillDir)
	result := skillManageResult{
		Success: true,
		Message: fmt.Sprintf("Skill %q created.", name),
		Path:    relPath,
	}
	if category != "" {
		result.Hint = fmt.Sprintf("To add reference files, templates, or scripts, use skill_manage(action='write_file', name='%s', file_path='references/example.md', file_content='...')", name)
	}
	return result
}

func (t *SkillManagerTool) handleEdit(in skillManageArgs) skillManageResult {
	name := strings.TrimSpace(in.Name)
	if err := validateSkillName(name); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	content := strings.TrimSpace(in.Content)
	if err := validateSkillContent(content); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	// Find existing skill
	skillPath := findSkill(t.cfg.Root, name)
	if skillPath == "" {
		return skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("Skill %q not found. Use skills_list() to see available skills.", name),
		}
	}

	skillMD := filepath.Join(skillPath, "SKILL.md")

	// Write new content
	if err := atomicWriteText(skillMD, content); err != nil {
		return skillManageResult{Success: false, Error: "failed to write SKILL.md: " + err.Error()}
	}

	return skillManageResult{
		Success: true,
		Message: fmt.Sprintf("Skill %q updated.", name),
		Path:    skillPath,
	}
}

func (t *SkillManagerTool) handlePatch(in skillManageArgs) skillManageResult {
	name := strings.TrimSpace(in.Name)
	if err := validateSkillName(name); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	oldString := strings.TrimSpace(in.OldString)
	if oldString == "" {
		return skillManageResult{Success: false, Error: "old_string is required for patch."}
	}

	newString := in.NewString // Can be empty to delete

	// Find existing skill
	skillPath := findSkill(t.cfg.Root, name)
	if skillPath == "" {
		return skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("Skill %q not found.", name),
		}
	}

	skillMD := filepath.Join(skillPath, "SKILL.md")

	// Read content
	content, err := os.ReadFile(skillMD)
	if err != nil {
		return skillManageResult{Success: false, Error: "failed to read SKILL.md: " + err.Error()}
	}
	contentStr := string(content)

	// Find old_string in content
	idx := strings.Index(contentStr, oldString)
	if idx < 0 {
		return skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("old_string not found in SKILL.md. Include enough context to ensure uniqueness."),
		}
	}

	// Check for multiple occurrences unless replace_all
	if !in.ReplaceAll {
		nextIdx := strings.Index(contentStr[idx+len(oldString):], oldString)
		if nextIdx >= 0 {
			return skillManageResult{
				Success: false,
				Error:   "Multiple matches found. Set replace_all=true to replace all occurrences.",
			}
		}
	}

	// Apply replacement
	var newContent string
	matchCount := 0
	if in.ReplaceAll {
		newContent = strings.ReplaceAll(contentStr, oldString, newString)
		matchCount = strings.Count(contentStr, oldString)
	} else {
		newContent = contentStr[:idx] + newString + contentStr[idx+len(oldString):]
		matchCount = 1
	}

	// Validate resulting content
	if err := validateSkillContent(newContent); err != "" {
		return skillManageResult{Success: false, Error: fmt.Sprintf("Patch would break SKILL.md structure: %s", err)}
	}

	// Write patched content
	if err := atomicWriteText(skillMD, newContent); err != nil {
		return skillManageResult{Success: false, Error: "failed to write patched SKILL.md: " + err.Error()}
	}

	return skillManageResult{
		Success: true,
		Message: fmt.Sprintf("Patched SKILL.md in skill %q (%d replacement%s).", name, matchCount, plural(matchCount)),
	}
}

func (t *SkillManagerTool) handleDelete(in skillManageArgs) skillManageResult {
	name := strings.TrimSpace(in.Name)
	if err := validateSkillName(name); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	// Find existing skill
	skillPath := findSkill(t.cfg.Root, name)
	if skillPath == "" {
		return skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("Skill %q not found.", name),
		}
	}

	// Remove skill directory
	if err := os.RemoveAll(skillPath); err != nil {
		return skillManageResult{Success: false, Error: "failed to delete skill: " + err.Error()}
	}

	// Clean up empty parent directories (up to but not including root/active)
	parent := filepath.Dir(skillPath)
	rootActive := filepath.Join(t.cfg.Root, "active")
	for parent != "" && parent != rootActive && parent != t.cfg.Root {
		entries, err := os.ReadDir(parent)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(parent)
		parent = filepath.Dir(parent)
	}

	return skillManageResult{
		Success: true,
		Message: fmt.Sprintf("Skill %q deleted.", name),
	}
}

// =============================================================================
// Skill discovery
// =============================================================================

// findSkill returns the path to a skill directory given its name, or "" if not found.
func findSkill(root, name string) string {
	if root == "" {
		return ""
	}
	activeDir := filepath.Join(root, "active")
	if _, err := os.Stat(activeDir); os.IsNotExist(err) {
		return ""
	}

	name = strings.ToLower(strings.TrimSpace(name))
	entries, err := os.ReadDir(activeDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		category := entry.Name()
		categoryPath := filepath.Join(activeDir, category)
		subEntries, err := os.ReadDir(categoryPath)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}
			if strings.ToLower(subEntry.Name()) == name {
				return filepath.Join(categoryPath, subEntry.Name())
			}
		}
		// Also check skills directly in active/ (no category)
		if strings.ToLower(category) == name {
			return categoryPath
		}
	}
	return ""
}

// resolveSkillDir returns the directory path for a skill.
func resolveSkillDir(root, name, category string) string {
	if category != "" {
		return filepath.Join(root, "active", category, name)
	}
	return filepath.Join(root, "active", name)
}

// =============================================================================
// File utilities
// =============================================================================

// atomicWriteText writes content to a file atomically.
func atomicWriteText(path string, content string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Write to temp file in same directory, then rename
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	defer os.Remove(tmpPath) // Best effort cleanup

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// =============================================================================
// Helpers
// =============================================================================

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func ensurePortableFrontmatter(content, name string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	idx := strings.Index(content[3:], "\n---")
	if idx < 0 {
		return content
	}
	fm := content[3 : 3+idx]
	rest := content[3+idx+len("\n---"):]

	has := func(field string) bool {
		return strings.Contains(fm, "\n"+field+":") || strings.HasPrefix(fm, field+":")
	}
	if !has("version") {
		fm += "\nversion: 1.0.0"
	}
	if !has("author") {
		fm += "\nauthor: gormes-agent"
	}
	if !has("date") {
		fm += "\ndate: " + time.Now().UTC().Format("2006-01-02")
	}
	return "---" + fm + "\n---" + rest
}
