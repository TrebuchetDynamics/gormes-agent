package skillprompt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"gopkg.in/yaml.v3"
)

const skillsSnapshotVersion = 1

// SkillsPromptOptions describes the native Hermes-compatible skill-index prompt
// builder inputs. Nil AvailableTools and AvailableToolsets preserve Hermes'
// backward-compatible show-all behavior; empty-but-non-nil slices are restrictive.
type SkillsPromptOptions struct {
	LocalDir           string
	ExternalDirs       []string
	SnapshotPath       string
	AvailableTools     []string
	AvailableToolsets  []string
	DisabledSkillNames []string
	Platform           string
}

type SkillsPromptEvidence struct {
	Code   string
	Path   string
	Reason string
}

type skillPromptEntry struct {
	SkillName       string                 `json:"skill_name"`
	Category        string                 `json:"category"`
	FrontmatterName string                 `json:"frontmatter_name"`
	Description     string                 `json:"description"`
	Platforms       []string               `json:"platforms,omitempty"`
	Conditions      skills.SkillConditions `json:"conditions"`
	Status          skills.SkillStatusCode `json:"-"`
	Reason          string                 `json:"-"`
}

type skillsPromptSnapshot struct {
	Version              int                 `json:"version"`
	Manifest             map[string][2]int64 `json:"manifest"`
	Skills               []skillPromptEntry  `json:"skills"`
	CategoryDescriptions map[string]string   `json:"category_descriptions"`
}

type categoryPrompt struct {
	Description string
	Skills      []skillPromptEntry
}

var skillsPromptCache = struct {
	sync.Mutex
	values map[string]string
}{values: map[string]string{}}

func ResetSkillsPromptCacheForTest() {
	skillsPromptCache.Lock()
	defer skillsPromptCache.Unlock()
	skillsPromptCache.values = map[string]string{}
}

func BuildSkillsSystemPrompt(opts SkillsPromptOptions) (string, []SkillsPromptEvidence, error) {
	localDir := filepath.Clean(opts.LocalDir)
	if strings.TrimSpace(opts.LocalDir) == "" {
		return "", nil, nil
	}
	cacheKey := skillsPromptCacheKey(opts)
	skillsPromptCache.Lock()
	if cached, ok := skillsPromptCache.values[cacheKey]; ok {
		skillsPromptCache.Unlock()
		return cached, nil, nil
	}
	skillsPromptCache.Unlock()

	var evidence []SkillsPromptEvidence
	categories := map[string]*categoryPrompt{}
	seen := map[string]bool{}

	loaded := false
	if opts.SnapshotPath != "" {
		manifest, err := buildSkillsManifest(localDir)
		if err != nil {
			return "", nil, err
		}
		if snapshot, ok := loadSkillsSnapshot(opts.SnapshotPath, manifest); ok {
			evidence = append(evidence, SkillsPromptEvidence{Code: "skills_prompt_snapshot_hit", Path: opts.SnapshotPath})
			for _, entry := range snapshot.Skills {
				if includeSkillEntry(entry, opts, &evidence) {
					addPromptEntry(categories, seen, entry)
				}
			}
			for cat, desc := range snapshot.CategoryDescriptions {
				ensureCategory(categories, cat).Description = desc
			}
			loaded = true
		} else {
			evidence = append(evidence, SkillsPromptEvidence{Code: "skills_prompt_snapshot_miss", Path: opts.SnapshotPath})
		}
	}

	if !loaded {
		entries, descriptions, scanEvidence, err := scanSkillsDir(localDir, opts)
		if err != nil {
			return "", nil, err
		}
		evidence = append(evidence, scanEvidence...)
		for _, entry := range entries {
			if includeSkillEntry(entry, opts, &evidence) {
				addPromptEntry(categories, seen, entry)
			}
		}
		for cat, desc := range descriptions {
			ensureCategory(categories, cat).Description = desc
		}
		if opts.SnapshotPath != "" {
			manifest, err := buildSkillsManifest(localDir)
			if err != nil {
				return "", nil, err
			}
			if err := writeSkillsSnapshot(opts.SnapshotPath, manifest, entries, descriptions); err != nil {
				evidence = append(evidence, SkillsPromptEvidence{Code: "skills_prompt_snapshot_write_failed", Path: opts.SnapshotPath, Reason: err.Error()})
			}
		}
	}

	for _, externalDir := range opts.ExternalDirs {
		entries, descriptions, scanEvidence, err := scanSkillsDir(externalDir, opts)
		if err != nil {
			return "", nil, err
		}
		evidence = append(evidence, scanEvidence...)
		for cat, desc := range descriptions {
			if ensureCategory(categories, cat).Description == "" {
				categories[cat].Description = desc
			}
		}
		for _, entry := range entries {
			if includeSkillEntry(entry, opts, &evidence) {
				addPromptEntry(categories, seen, entry)
			}
		}
	}

	prompt := renderSkillsPrompt(categories)
	skillsPromptCache.Lock()
	skillsPromptCache.values[cacheKey] = prompt
	skillsPromptCache.Unlock()
	return prompt, evidence, nil
}

func skillsPromptCacheKey(opts SkillsPromptOptions) string {
	parts := []string{
		filepath.Clean(opts.LocalDir),
		strings.Join(sortedStrings(opts.ExternalDirs), "\x00"),
		strings.Join(sortedStrings(opts.AvailableTools), "\x00"),
		strings.Join(sortedStrings(opts.AvailableToolsets), "\x00"),
		strings.Join(sortedStrings(opts.DisabledSkillNames), "\x00"),
		strings.ToLower(strings.TrimSpace(opts.Platform)),
		fmt.Sprintf("toolsNil=%t", opts.AvailableTools == nil),
		fmt.Sprintf("toolsetsNil=%t", opts.AvailableToolsets == nil),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\xff")))
	return hex.EncodeToString(sum[:])
}

func scanSkillsDir(root string, opts SkillsPromptOptions) ([]skillPromptEntry, map[string]string, []SkillsPromptEvidence, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, nil, fmt.Errorf("skills prompt: %q is not a directory", root)
	}
	var skillPaths []string
	var descPaths []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Base(path) {
		case "SKILL.md":
			skillPaths = append(skillPaths, path)
		case "DESCRIPTION.md":
			descPaths = append(descPaths, path)
		}
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}
	sort.Strings(skillPaths)
	sort.Strings(descPaths)

	entries := make([]skillPromptEntry, 0, len(skillPaths))
	descriptions := map[string]string{}
	var evidence []SkillsPromptEvidence
	for _, path := range skillPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, nil, err
		}
		expectedSlug := filepath.Base(filepath.Dir(path))
		if errs := skills.ValidateSkillFrontmatter(raw, skills.FrontmatterValidateOptions{ExpectedSlug: expectedSlug}); len(errs) > 0 {
			evidence = append(evidence, SkillsPromptEvidence{Code: string(skills.SkillStatusFrontmatterInvalid), Path: path, Reason: "frontmatter validation failed"})
			continue
		}
		skill, err := skills.Parse(raw, skills.DefaultMaxDocumentBytes)
		if err != nil {
			evidence = append(evidence, SkillsPromptEvidence{Code: string(skills.SkillStatusFrontmatterInvalid), Path: path, Reason: err.Error()})
			continue
		}
		skill.Path = path
		entry := newSkillPromptEntry(root, path, skill)
		entries = append(entries, entry)
	}
	for _, path := range descPaths {
		desc, cat, err := readCategoryDescription(root, path)
		if err != nil {
			continue
		}
		if desc != "" {
			descriptions[cat] = desc
		}
	}
	return entries, descriptions, evidence, nil
}

func newSkillPromptEntry(root, path string, skill skills.Skill) skillPromptEntry {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	category := "general"
	skillName := filepath.Base(filepath.Dir(path))
	if len(parts) >= 2 {
		skillName = parts[len(parts)-2]
		if len(parts) > 2 {
			category = strings.Join(parts[:len(parts)-2], "/")
		} else {
			category = parts[0]
		}
	}
	name := strings.TrimSpace(skill.Name)
	if name == "" {
		name = skillName
	}
	return skillPromptEntry{SkillName: skillName, Category: category, FrontmatterName: name, Description: strings.TrimSpace(skill.Description), Platforms: skill.Platforms, Conditions: skill.Conditions}
}

func includeSkillEntry(entry skillPromptEntry, opts SkillsPromptOptions, evidence *[]SkillsPromptEvidence) bool {
	name := strings.TrimSpace(entry.FrontmatterName)
	if name == "" {
		name = entry.SkillName
	}
	disabled := stringBoolSet(opts.DisabledSkillNames)
	if disabled[name] || disabled[entry.SkillName] || disabled[strings.ToLower(name)] || disabled[strings.ToLower(entry.SkillName)] {
		*evidence = append(*evidence, SkillsPromptEvidence{Code: string(skills.SkillStatusDisabled), Path: entry.SkillName, Reason: "skill disabled"})
		return false
	}
	if !skillEntryMatchesPlatform(entry, opts.Platform) {
		*evidence = append(*evidence, SkillsPromptEvidence{Code: string(skills.SkillStatusUnsupported), Path: entry.SkillName, Reason: "unsupported platform"})
		return false
	}
	if !skillConditionsMatch(entry.Conditions, opts.AvailableTools, opts.AvailableToolsets) {
		*evidence = append(*evidence, SkillsPromptEvidence{Code: string(skills.SkillStatusConditionExcluded), Path: entry.SkillName, Reason: "condition excluded"})
		return false
	}
	return true
}

func skillConditionsMatch(conditions skills.SkillConditions, availableTools, availableToolsets []string) bool {
	if availableTools == nil && availableToolsets == nil {
		return true
	}
	tools := stringBoolSet(availableTools)
	toolsets := stringBoolSet(availableToolsets)
	for _, toolset := range conditions.FallbackForToolsets {
		if toolsets[toolset] {
			return false
		}
	}
	for _, tool := range conditions.FallbackForTools {
		if tools[tool] {
			return false
		}
	}
	for _, toolset := range conditions.RequiresToolsets {
		if !toolsets[toolset] {
			return false
		}
	}
	for _, tool := range conditions.RequiresTools {
		if !tools[tool] {
			return false
		}
	}
	return true
}

func skillEntryMatchesPlatform(entry skillPromptEntry, platform string) bool {
	if len(entry.Platforms) == 0 {
		return true
	}
	current := normalizePromptPlatform(platform)
	for _, allowed := range entry.Platforms {
		if current == normalizePromptPlatform(allowed) {
			return true
		}
	}
	return false
}

func normalizePromptPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "":
		return "linux"
	case "darwin", "mac", "macos", "osx":
		return "macos"
	case "windows", "win", "win32":
		return "windows"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func addPromptEntry(categories map[string]*categoryPrompt, seen map[string]bool, entry skillPromptEntry) {
	name := entry.FrontmatterName
	if name == "" {
		name = entry.SkillName
	}
	if seen[name] {
		return
	}
	seen[name] = true
	cat := ensureCategory(categories, entry.Category)
	cat.Skills = append(cat.Skills, entry)
}

func ensureCategory(categories map[string]*categoryPrompt, category string) *categoryPrompt {
	category = strings.TrimSpace(category)
	if category == "" {
		category = "general"
	}
	if categories[category] == nil {
		categories[category] = &categoryPrompt{}
	}
	return categories[category]
}

func renderSkillsPrompt(categories map[string]*categoryPrompt) string {
	if len(categories) == 0 {
		return ""
	}
	categoryNames := make([]string, 0, len(categories))
	for cat, data := range categories {
		if len(data.Skills) > 0 {
			categoryNames = append(categoryNames, cat)
		}
	}
	sort.Strings(categoryNames)
	if len(categoryNames) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines,
		"## Skills (mandatory)",
		"Before replying, scan the skills below. If a skill matches or is even partially relevant to your task, you MUST load it with skill_view(name) and follow its instructions.",
		"",
		"<available_skills>",
	)
	for _, cat := range categoryNames {
		data := categories[cat]
		sort.SliceStable(data.Skills, func(i, j int) bool {
			return data.Skills[i].FrontmatterName < data.Skills[j].FrontmatterName
		})
		if data.Description != "" {
			lines = append(lines, "  "+cat+": "+data.Description)
		} else {
			lines = append(lines, "  "+cat+":")
		}
		localSeen := map[string]bool{}
		for _, entry := range data.Skills {
			name := strings.TrimSpace(entry.FrontmatterName)
			if name == "" {
				name = entry.SkillName
			}
			if localSeen[name] {
				continue
			}
			localSeen[name] = true
			if entry.Description != "" {
				lines = append(lines, "    - "+name+": "+entry.Description)
			} else {
				lines = append(lines, "    - "+name)
			}
		}
	}
	lines = append(lines, "</available_skills>")
	return strings.Join(lines, "\n")
}

func buildSkillsManifest(root string) (map[string][2]int64, error) {
	manifest := map[string][2]int64{}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return manifest, nil
	}
	for _, filename := range []string{"SKILL.md", "DESCRIPTION.md"} {
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Base(path) != filename {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			manifest[filepath.ToSlash(rel)] = [2]int64{info.ModTime().UnixNano(), info.Size()}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return manifest, nil
}

func loadSkillsSnapshot(path string, manifest map[string][2]int64) (skillsPromptSnapshot, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return skillsPromptSnapshot{}, false
	}
	var snapshot skillsPromptSnapshot
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return skillsPromptSnapshot{}, false
	}
	if snapshot.Version != skillsSnapshotVersion {
		return skillsPromptSnapshot{}, false
	}
	if !sameManifest(snapshot.Manifest, manifest) {
		return skillsPromptSnapshot{}, false
	}
	return snapshot, true
}

func writeSkillsSnapshot(path string, manifest map[string][2]int64, entries []skillPromptEntry, descriptions map[string]string) error {
	payload := skillsPromptSnapshot{Version: skillsSnapshotVersion, Manifest: manifest, Skills: entries, CategoryDescriptions: descriptions}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sameManifest(a, b map[string][2]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		if bv, ok := b[key]; !ok || bv != av {
			return false
		}
	}
	return true
}

func readCategoryDescription(root, path string) (string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	frontmatter, err := parsePromptFrontmatter(b)
	if err != nil {
		return "", "", err
	}
	desc, _ := frontmatter["description"].(string)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	cat := "general"
	if len(parts) > 1 {
		cat = strings.Join(parts[:len(parts)-1], "/")
	}
	return strings.Trim(strings.TrimSpace(desc), "'\""), cat, nil
}

func parsePromptFrontmatter(raw []byte) (map[string]any, error) {
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		return nil, nil
	}
	rest := text[len("---\n"):]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, fmt.Errorf("frontmatter close marker missing")
	}
	var out map[string]any
	if err := yaml.Unmarshal([]byte(rest[:idx]), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func stringBoolSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}
