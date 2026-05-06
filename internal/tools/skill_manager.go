package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	Root              string
	Timeout           time.Duration
	WriteOrigin       string
	GuardAgentCreated bool
	GuardScanner      func(skillDir string) error
}

// NewSkillManagerTool returns a skill management tool.
func NewSkillManagerTool(cfg SkillManagerToolConfig) Tool {
	return &SkillManagerTool{cfg: normalizeSkillManagerConfig(cfg)}
}

type SkillManagerTool struct {
	cfg SkillManagerToolConfig
}

type skillManageArgs struct {
	Action       string `json:"action"`
	Name         string `json:"name"`
	Content      string `json:"content"`
	Category     string `json:"category"`
	FilePath     string `json:"file_path"`
	FileContent  string `json:"file_content"`
	OldString    string `json:"old_string"`
	NewString    string `json:"new_string"`
	ReplaceAll   bool   `json:"replace_all"`
	AbsorbedInto string `json:"absorbed_into"`
}

type skillManageResult struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message,omitempty"`
	Path           string   `json:"path,omitempty"`
	Error          string   `json:"error,omitempty"`
	Hint           string   `json:"hint,omitempty"`
	AvailableFiles []string `json:"available_files,omitempty"`
}

const (
	maxSkillNameLength   = 64
	maxDescriptionLength = 1024
	maxContentChars      = 100_000
	maxSkillFileBytes    = 1_048_576
	validNameRE          = `^[a-z0-9][a-z0-9._-]*$`
)

var nameRegex = regexp.MustCompile(validNameRE)

const (
	SkillWriteOriginForeground       = "foreground"
	SkillWriteOriginBackgroundReview = "background_review"
)

var allowedSkillSupportDirs = map[string]bool{
	"assets":     true,
	"references": true,
	"scripts":    true,
	"templates":  true,
}

// =============================================================================
// Tool interface
// =============================================================================

func (*SkillManagerTool) Name() string { return SkillManagerToolName }

func (*SkillManagerTool) Description() string {
	return "Manage skills (create, update, delete). Skills are your procedural memory — reusable approaches for recurring task types. Actions: create, edit, patch, delete, write_file, remove_file."
}

func (*SkillManagerTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "patch", "edit", "delete", "write_file", "remove_file"],
				"description": "The action to perform: create, patch, edit, delete, write_file, or remove_file."
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
			"file_path": {
				"type": "string",
				"description": "Supporting file path under references/, templates/, scripts/, or assets/. Used by write_file, remove_file, and support-file patch."
			},
			"file_content": {
				"type": "string",
				"description": "Content for write_file. Empty content is allowed."
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
			},
			"absorbed_into": {
				"type": "string",
				"description": "For delete: target skill name when content was consolidated into another skill, or an empty string when intentionally pruned."
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
	case "write_file":
		return json.Marshal(t.handleWriteFile(in))
	case "remove_file":
		return json.Marshal(t.handleRemoveFile(in))
	default:
		return json.Marshal(skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown action %q. Use: create, edit, patch, delete, write_file, remove_file", in.Action),
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
	if cfg.WriteOrigin == "" {
		cfg.WriteOrigin = SkillWriteOriginForeground
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

func validateSupportFilePath(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "file_path is required."
	}
	if filepath.IsAbs(filePath) {
		return "absolute support-file paths are not allowed."
	}
	cleaned := filepath.Clean(filePath)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "file_path must include a file name under references/, templates/, scripts/, or assets/."
	}
	parts := strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(parts) == 0 {
		return `file_path must include a file name under references/, templates/, scripts/, or assets/. Example: "references/example.md".`
	}
	if len(parts) < 2 {
		return fmt.Sprintf("Provide a file path, not just a directory. Example: %q.", filepath.Join(parts[0], "example.md"))
	}
	for _, part := range parts {
		if part == ".." {
			return "path traversal is not allowed in support-file paths."
		}
	}
	if !allowedSkillSupportDirs[parts[0]] {
		return "support files must be under references/, templates/, scripts/, or assets/."
	}
	return ""
}

func supportFileTarget(skillDir, filePath string) (string, string) {
	if err := validateSupportFilePath(filePath); err != "" {
		return "", err
	}
	target := filepath.Join(skillDir, filepath.Clean(filePath))
	if err := validatePathWithinSkillDir(skillDir, target); err != "" {
		return "", err
	}
	return target, ""
}

func validatePathWithinSkillDir(skillDir, target string) string {
	rootAbs, err := filepath.Abs(skillDir)
	if err != nil {
		return err.Error()
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err.Error()
	}
	if !pathWithin(rootAbs, targetAbs) {
		return "support file path escapes skill directory."
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		rootReal = rootAbs
	}
	ancestor := deepestExistingAncestor(targetAbs)
	if ancestor != "" {
		ancestorReal, err := filepath.EvalSymlinks(ancestor)
		if err == nil && !pathWithin(rootReal, ancestorReal) {
			return "support file path escapes skill directory through a symlink."
		}
	}
	return ""
}

func deepestExistingAncestor(path string) string {
	for current := path; current != "" && current != "."; current = filepath.Dir(current) {
		if _, err := os.Lstat(current); err == nil {
			return current
		}
		next := filepath.Dir(current)
		if next == current {
			return ""
		}
	}
	return ""
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func supportFileList(skillDir string) []string {
	var out []string
	for dir := range allowedSkillSupportDirs {
		root := filepath.Join(skillDir, dir)
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(skillDir, path)
			if relErr == nil {
				out = append(out, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

func (t *SkillManagerTool) pinnedRefusal(name string) *skillManageResult {
	pinned, err := skills.IsPinned(t.cfg.Root, name)
	if err != nil || !pinned {
		return nil
	}
	return &skillManageResult{
		Success: false,
		Error:   fmt.Sprintf("Skill %q is pinned and cannot be modified by skill_manage.", name),
		Hint:    fmt.Sprintf("Ask the user to run `gormes curator unpin %s` if they want this skill changed.", name),
	}
}

func (t *SkillManagerTool) scanSkill(skillDir string) string {
	if !t.cfg.GuardAgentCreated || t.cfg.GuardScanner == nil {
		return ""
	}
	if err := t.cfg.GuardScanner(skillDir); err != nil {
		return err.Error()
	}
	return ""
}

func restoreSkillFile(path string, original []byte, existed bool) {
	if existed {
		_ = atomicWriteText(path, string(original))
		return
	}
	_ = os.Remove(path)
}

func (t *SkillManagerTool) recordSuccessfulMutation(action, name string) {
	switch action {
	case "create":
		if t.cfg.WriteOrigin == SkillWriteOriginBackgroundReview {
			_ = skills.MarkAgentCreated(t.cfg.Root, name)
		}
	case "edit", "patch", "write_file", "remove_file":
		_ = skills.BumpPatch(t.cfg.Root, name)
	case "delete":
		_ = skills.ForgetUsageRecord(t.cfg.Root, name)
	}
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
	if scanErr := t.scanSkill(skillDir); scanErr != "" {
		os.RemoveAll(skillDir)
		return skillManageResult{Success: false, Error: "skill guard blocked create: " + scanErr}
	}

	relPath, _ := filepath.Rel(t.cfg.Root, skillDir)
	t.recordSuccessfulMutation("create", name)
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
	if refusal := t.pinnedRefusal(name); refusal != nil {
		return *refusal
	}

	skillMD := filepath.Join(skillPath, "SKILL.md")
	original, readErr := os.ReadFile(skillMD)
	existed := readErr == nil

	// Write new content
	if err := atomicWriteText(skillMD, content); err != nil {
		return skillManageResult{Success: false, Error: "failed to write SKILL.md: " + err.Error()}
	}
	if scanErr := t.scanSkill(skillPath); scanErr != "" {
		restoreSkillFile(skillMD, original, existed)
		return skillManageResult{Success: false, Error: "skill guard blocked edit: " + scanErr}
	}
	t.recordSuccessfulMutation("edit", name)

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
	if refusal := t.pinnedRefusal(name); refusal != nil {
		return *refusal
	}

	targetLabel := "SKILL.md"
	targetPath := filepath.Join(skillPath, "SKILL.md")
	if strings.TrimSpace(in.FilePath) != "" {
		var errText string
		targetPath, errText = supportFileTarget(skillPath, in.FilePath)
		if errText != "" {
			return skillManageResult{Success: false, Error: errText}
		}
		targetLabel = filepath.ToSlash(filepath.Clean(in.FilePath))
	}

	// Read content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return skillManageResult{Success: false, Error: "failed to read " + targetLabel + ": " + err.Error()}
	}
	contentStr := string(content)

	// Find old_string in content
	idx := strings.Index(contentStr, oldString)
	if idx < 0 {
		return skillManageResult{
			Success: false,
			Error:   fmt.Sprintf("old_string not found in %s. Include enough context to ensure uniqueness.", targetLabel),
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
	if targetLabel == "SKILL.md" {
		if err := validateSkillContent(newContent); err != "" {
			return skillManageResult{Success: false, Error: fmt.Sprintf("Patch would break SKILL.md structure: %s", err)}
		}
	}
	if len([]byte(newContent)) > maxSkillFileBytes {
		return skillManageResult{Success: false, Error: fmt.Sprintf("%s content exceeds %d bytes.", targetLabel, maxSkillFileBytes)}
	}

	// Write patched content
	if err := atomicWriteText(targetPath, newContent); err != nil {
		return skillManageResult{Success: false, Error: "failed to write patched " + targetLabel + ": " + err.Error()}
	}
	if scanErr := t.scanSkill(skillPath); scanErr != "" {
		restoreSkillFile(targetPath, content, true)
		return skillManageResult{Success: false, Error: "skill guard blocked patch: " + scanErr}
	}
	t.recordSuccessfulMutation("patch", name)

	return skillManageResult{
		Success: true,
		Message: fmt.Sprintf("Patched %s in skill %q (%d replacement%s).", targetLabel, name, matchCount, plural(matchCount)),
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
	if refusal := t.pinnedRefusal(name); refusal != nil {
		return *refusal
	}

	absorbedInto := strings.TrimSpace(in.AbsorbedInto)
	if absorbedInto != "" {
		if absorbedInto == name {
			return skillManageResult{Success: false, Error: fmt.Sprintf("absorbed_into=%q cannot equal the skill being deleted.", absorbedInto)}
		}
		if findSkill(t.cfg.Root, absorbedInto) == "" {
			return skillManageResult{Success: false, Error: fmt.Sprintf("absorbed_into=%q does not exist. Create or patch the umbrella skill first, then retry the delete.", absorbedInto)}
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

	t.recordSuccessfulMutation("delete", name)
	message := fmt.Sprintf("Skill %q deleted.", name)
	if absorbedInto != "" {
		message += fmt.Sprintf(" Content absorbed into %q.", absorbedInto)
	}
	return skillManageResult{
		Success: true,
		Message: message,
	}
}

func (t *SkillManagerTool) handleWriteFile(in skillManageArgs) skillManageResult {
	name := strings.TrimSpace(in.Name)
	if err := validateSkillName(name); err != "" {
		return skillManageResult{Success: false, Error: err}
	}

	skillPath := findSkill(t.cfg.Root, name)
	if skillPath == "" {
		return skillManageResult{Success: false, Error: fmt.Sprintf("Skill %q not found. Create it first with action='create'.", name)}
	}
	if refusal := t.pinnedRefusal(name); refusal != nil {
		return *refusal
	}
	if len([]byte(in.FileContent)) > maxSkillFileBytes {
		return skillManageResult{Success: false, Error: fmt.Sprintf("file_content exceeds %d bytes.", maxSkillFileBytes)}
	}
	targetPath, errText := supportFileTarget(skillPath, in.FilePath)
	if errText != "" {
		return skillManageResult{Success: false, Error: errText}
	}
	original, readErr := os.ReadFile(targetPath)
	existed := readErr == nil
	if err := atomicWriteText(targetPath, in.FileContent); err != nil {
		return skillManageResult{Success: false, Error: "failed to write support file: " + err.Error()}
	}
	if scanErr := t.scanSkill(skillPath); scanErr != "" {
		restoreSkillFile(targetPath, original, existed)
		return skillManageResult{Success: false, Error: "skill guard blocked write_file: " + scanErr}
	}
	t.recordSuccessfulMutation("write_file", name)
	rel, _ := filepath.Rel(skillPath, targetPath)
	return skillManageResult{
		Success: true,
		Message: fmt.Sprintf("File %q written to skill %q.", filepath.ToSlash(rel), name),
		Path:    targetPath,
	}
}

func (t *SkillManagerTool) handleRemoveFile(in skillManageArgs) skillManageResult {
	name := strings.TrimSpace(in.Name)
	if err := validateSkillName(name); err != "" {
		return skillManageResult{Success: false, Error: err}
	}
	skillPath := findSkill(t.cfg.Root, name)
	if skillPath == "" {
		return skillManageResult{Success: false, Error: fmt.Sprintf("Skill %q not found.", name)}
	}
	if refusal := t.pinnedRefusal(name); refusal != nil {
		return *refusal
	}
	targetPath, errText := supportFileTarget(skillPath, in.FilePath)
	if errText != "" {
		return skillManageResult{Success: false, Error: errText}
	}
	if _, err := os.Stat(targetPath); err != nil {
		return skillManageResult{
			Success:        false,
			Error:          fmt.Sprintf("File %q not found in skill %q.", filepath.ToSlash(filepath.Clean(in.FilePath)), name),
			AvailableFiles: supportFileList(skillPath),
		}
	}
	original, _ := os.ReadFile(targetPath)
	if err := os.Remove(targetPath); err != nil {
		return skillManageResult{Success: false, Error: "failed to remove support file: " + err.Error()}
	}
	if scanErr := t.scanSkill(skillPath); scanErr != "" {
		restoreSkillFile(targetPath, original, true)
		return skillManageResult{Success: false, Error: "skill guard blocked remove_file: " + scanErr}
	}
	parent := filepath.Dir(targetPath)
	if parent != skillPath {
		if entries, err := os.ReadDir(parent); err == nil && len(entries) == 0 {
			_ = os.Remove(parent)
		}
	}
	t.recordSuccessfulMutation("remove_file", name)
	return skillManageResult{
		Success: true,
		Message: fmt.Sprintf("File %q removed from skill %q.", filepath.ToSlash(filepath.Clean(in.FilePath)), name),
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
