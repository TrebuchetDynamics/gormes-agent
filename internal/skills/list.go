package skills

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SkillStatusEnabled SkillStatusCode = "enabled"
)

type SkillRow struct {
	Name     string
	Category string
	Source   string
	Trust    string
	Path     string
	Status   SkillStatusCode
}

type ListOptions struct {
	Source      string
	EnabledOnly bool
}

type skillListMeta struct {
	Category string `json:"category"`
	Source   string `json:"source"`
	Trust    string `json:"trust"`
}

func ListInstalledSkills(opts ListOptions, disabled map[string]struct{}) []SkillRow {
	rows := installedSkillRows()
	source := normalizedListSource(opts.Source)
	disabledNames := normalizedDisabledSet(disabled)

	out := make([]SkillRow, 0, len(rows))
	for _, row := range rows {
		if source != "all" && row.Source != source {
			continue
		}
		if _, ok := disabledNames[strings.ToLower(strings.TrimSpace(row.Name))]; ok {
			row.Status = SkillStatusDisabled
		} else if row.Status == "" {
			row.Status = SkillStatusEnabled
		}
		if opts.EnabledOnly && row.Status != SkillStatusEnabled {
			continue
		}
		out = append(out, row)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func installedSkillRows() []SkillRow {
	store := NewStore(defaultSkillsRoot(), 0)
	snapshot, err := store.SnapshotActive()
	if err != nil {
		return nil
	}

	rows := make([]SkillRow, 0, len(snapshot.Skills))
	for _, skill := range snapshot.Skills {
		rows = append(rows, activeSkillRow(store.ActiveDir(), skill))
	}
	rows = append(rows, bundledSkillRows()...)
	return rows
}

func activeSkillRow(activeDir string, skill Skill) SkillRow {
	row := baseSkillRow(skill)
	meta := readSkillListMeta(skill.Path)
	row.Category = firstNonBlank(meta.Category, categoryFromSkillPath(activeDir, skill.Path))
	row.Source = normalizeInstalledSource(firstNonBlank(meta.Source, sourceFromSkillPath(activeDir, skill.Path)))
	row.Trust = firstNonBlank(meta.Trust, defaultTrustForSource(row.Source))
	return row
}

func bundledSkillRows() []SkillRow {
	root := bundledSkillsRoot()
	if root == "" {
		return nil
	}
	skills, err := loadSkillDocsFromDir(root, DefaultMaxDocumentBytes)
	if err != nil {
		return nil
	}

	rows := make([]SkillRow, 0, len(skills))
	for _, skill := range skills {
		row := baseSkillRow(skill)
		meta := readSkillListMeta(skill.Path)
		row.Category = firstNonBlank(meta.Category, bundledCategoryFromSkillPath(root, skill.Path))
		row.Source = normalizeInstalledSource(firstNonBlank(meta.Source, "builtin"))
		row.Trust = firstNonBlank(meta.Trust, "system")
		rows = append(rows, row)
	}
	return rows
}

func baseSkillRow(skill Skill) SkillRow {
	row := SkillRow{
		Name: skill.Name,
		Path: skill.Path,
	}
	if len(missingSkillCredentials(skill, nil)) > 0 {
		row.Status = SkillStatusMissingPrerequisite
	} else {
		row.Status = SkillStatusEnabled
	}
	return row
}

func loadSkillDocsFromDir(root string, maxBytes int) ([]Skill, error) {
	info, err := os.Stat(root)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	case !info.IsDir():
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
		return nil, err
	}
	sort.Strings(paths)

	out := make([]Skill, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		skill, err := Parse(raw, maxBytes)
		if err != nil {
			return nil, err
		}
		skill.Path = path
		out = append(out, skill)
	}
	return out, nil
}

func bundledSkillsRoot() string {
	if root := strings.TrimSpace(os.Getenv("GORMES_BUNDLED_SKILLS_ROOT")); root != "" {
		return root
	}
	if strings.TrimSpace(os.Getenv("GORMES_SKILLS_ROOT")) != "" {
		return ""
	}
	return findBundledSkillsRoot()
}

func findBundledSkillsRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func readSkillListMeta(skillPath string) skillListMeta {
	if skillPath == "" {
		return skillListMeta{}
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(skillPath), "meta.json"))
	if err != nil {
		return skillListMeta{}
	}
	var meta skillListMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return skillListMeta{}
	}
	return meta
}

func defaultSkillsRoot() string {
	if root := strings.TrimSpace(os.Getenv("GORMES_SKILLS_ROOT")); root != "" {
		return root
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "gormes", "skills")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".local", "share", "gormes", "skills")
	}
	return filepath.Join(home, ".local", "share", "gormes", "skills")
}

func categoryFromSkillPath(activeDir, skillPath string) string {
	relDir := relativeSkillDir(activeDir, skillPath)
	if relDir == "" || relDir == "." {
		return ""
	}
	parts := splitPath(relDir)
	if len(parts) <= 1 {
		return ""
	}
	return filepath.Join(parts[:len(parts)-1]...)
}

func bundledCategoryFromSkillPath(root, skillPath string) string {
	if root == "" || skillPath == "" {
		return ""
	}
	rel, err := filepath.Rel(root, filepath.Dir(skillPath))
	if err != nil {
		return ""
	}
	parts := splitPath(rel)
	if len(parts) <= 1 {
		return ""
	}
	return filepath.Join(parts[:len(parts)-1]...)
}

func sourceFromSkillPath(activeDir, skillPath string) string {
	parts := splitPath(relativeSkillDir(activeDir, skillPath))
	if len(parts) > 1 {
		return parts[0]
	}
	return "local"
}

func relativeSkillDir(activeDir, skillPath string) string {
	if activeDir == "" || skillPath == "" {
		return ""
	}
	rel, err := filepath.Rel(activeDir, filepath.Dir(skillPath))
	if err != nil {
		return ""
	}
	return rel
}

func splitPath(path string) []string {
	if path == "" || path == "." {
		return nil
	}
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == filepath.Separator
	})
	out := parts[:0]
	for _, part := range parts {
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}

func normalizedDisabledSet(disabled map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(disabled))
	for name := range disabled {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}

func normalizedListSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return "all"
	}
	switch source {
	case "all", "hub", "builtin", "local":
		return source
	default:
		return source
	}
}

func normalizeInstalledSource(source string) string {
	source = normalizedListSource(source)
	switch source {
	case "hub", "builtin", "local":
		return source
	default:
		return "local"
	}
}

func defaultTrustForSource(source string) string {
	switch source {
	case "hub":
		return "community"
	case "builtin":
		return "builtin"
	default:
		return "local"
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
