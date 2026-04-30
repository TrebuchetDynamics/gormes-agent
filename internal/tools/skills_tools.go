package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

const (
	SkillsListToolName = "skills_list"
	SkillViewToolName  = "skill_view"
)

const skillsToolDefaultTimeout = 10 * time.Second

// SkillsToolsConfig configures the read-only skills tool surface.
type SkillsToolsConfig struct {
	Root             string
	BundledRoot      string
	MaxDocumentBytes int
	Timeout          time.Duration
}

// NewSkillsTools returns the built-in Hermes-compatible read-only skills tools.
func NewSkillsTools(cfg SkillsToolsConfig) []Tool {
	return []Tool{
		NewSkillsListTool(cfg),
		NewSkillViewTool(cfg),
	}
}

type SkillsListTool struct {
	cfg SkillsToolsConfig
}

type SkillViewTool struct {
	cfg SkillsToolsConfig
}

type skillsListToolArgs struct {
	Category string `json:"category"`
}

type skillViewToolArgs struct {
	Name     string `json:"name"`
	FilePath string `json:"file_path"`
}

type skillsListToolResult struct {
	Success    bool                  `json:"success"`
	Skills     []skillsListToolSkill `json:"skills,omitempty"`
	Categories []string              `json:"categories,omitempty"`
	Count      int                   `json:"count"`
	Message    string                `json:"message,omitempty"`
	Hint       string                `json:"hint,omitempty"`
	Error      string                `json:"error,omitempty"`
}

type skillsListToolSkill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Source      string `json:"source,omitempty"`
	Trust       string `json:"trust,omitempty"`
	Status      string `json:"status,omitempty"`
}

type skillViewToolResult struct {
	Success       bool                `json:"success"`
	Name          string              `json:"name,omitempty"`
	Description   string              `json:"description,omitempty"`
	Tags          []string            `json:"tags,omitempty"`
	RelatedSkills []string            `json:"related_skills,omitempty"`
	Content       string              `json:"content,omitempty"`
	Path          string              `json:"path,omitempty"`
	SkillDir      string              `json:"skill_dir,omitempty"`
	LinkedFiles   map[string][]string `json:"linked_files,omitempty"`
	UsageHint     string              `json:"usage_hint,omitempty"`
	File          string              `json:"file,omitempty"`
	FileType      string              `json:"file_type,omitempty"`
	IsBinary      bool                `json:"is_binary,omitempty"`

	ReadinessStatus string              `json:"readiness_status,omitempty"`
	AvailableSkills []string            `json:"available_skills,omitempty"`
	AvailableFiles  map[string][]string `json:"available_files,omitempty"`
	Hint            string              `json:"hint,omitempty"`
	Error           string              `json:"error,omitempty"`
}

type skillToolDoc struct {
	Skill    skills.Skill
	Category string
	Source   string
	Trust    string
	Status   string
}

type skillToolMeta struct {
	Category string `json:"category"`
	Source   string `json:"source"`
	Trust    string `json:"trust"`
}

func NewSkillsListTool(cfg SkillsToolsConfig) *SkillsListTool {
	return &SkillsListTool{cfg: normalizeSkillsToolsConfig(cfg)}
}

func (*SkillsListTool) Name() string { return SkillsListToolName }

func (*SkillsListTool) Description() string {
	return "List available skills (name + description). Use skill_view(name) to load full content."
}

func (*SkillsListTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"category":{"type":"string","description":"Optional category filter to narrow results."}}}`)
}

func (t *SkillsListTool) Timeout() time.Duration {
	if t == nil || t.cfg.Timeout <= 0 {
		return skillsToolDefaultTimeout
	}
	return t.cfg.Timeout
}

func (t *SkillsListTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	var in skillsListToolArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return json.Marshal(skillsListToolResult{Success: false, Error: "invalid skills_list args: " + err.Error()})
	}

	docs, err := loadSkillToolDocs(t.cfg)
	if err != nil {
		return json.Marshal(skillsListToolResult{Success: false, Error: "load skills: " + err.Error()})
	}

	category := strings.ToLower(strings.TrimSpace(in.Category))
	rows := make([]skillsListToolSkill, 0, len(docs))
	categories := map[string]struct{}{}
	for _, doc := range docs {
		if category != "" && strings.ToLower(doc.Category) != category {
			continue
		}
		if doc.Category != "" {
			categories[doc.Category] = struct{}{}
		}
		rows = append(rows, skillsListToolSkill{
			Name:        doc.Skill.Name,
			Description: doc.Skill.Description,
			Category:    doc.Category,
			Source:      doc.Source,
			Trust:       doc.Trust,
			Status:      doc.Status,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Category != rows[j].Category {
			return rows[i].Category < rows[j].Category
		}
		return rows[i].Name < rows[j].Name
	})
	categoryList := sortedSkillToolCategories(categories)
	return json.Marshal(skillsListToolResult{
		Success:    true,
		Skills:     rows,
		Categories: categoryList,
		Count:      len(rows),
		Hint:       "Use skill_view(name) to see full content, tags, and linked files",
	})
}

func NewSkillViewTool(cfg SkillsToolsConfig) *SkillViewTool {
	return &SkillViewTool{cfg: normalizeSkillsToolsConfig(cfg)}
}

func (*SkillViewTool) Name() string { return SkillViewToolName }

func (*SkillViewTool) Description() string {
	return "Skills allow for loading information about specific tasks and workflows, as well as scripts and templates. Load a skill's full content or access its linked files (references, templates, scripts). First call returns SKILL.md content plus a linked_files dict showing available references/templates/scripts. To access those, call again with file_path parameter."
}

func (*SkillViewTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Skill name to load."},"file_path":{"type":"string","description":"Optional relative path to a linked file inside the skill directory."}},"required":["name"]}`)
}

func (t *SkillViewTool) Timeout() time.Duration {
	if t == nil || t.cfg.Timeout <= 0 {
		return skillsToolDefaultTimeout
	}
	return t.cfg.Timeout
}

func (t *SkillViewTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	var in skillViewToolArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return json.Marshal(skillViewError("invalid skill_view args: " + err.Error()))
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return json.Marshal(skillViewError("name is required"))
	}

	docs, err := loadSkillToolDocs(t.cfg)
	if err != nil {
		return json.Marshal(skillViewError("load skills: " + err.Error()))
	}
	doc, ok := findSkillToolDoc(docs, name)
	if !ok {
		return json.Marshal(skillViewToolResult{
			Success:         false,
			Error:           fmt.Sprintf("Skill %q not found.", name),
			AvailableSkills: firstSkillToolNames(docs, 20),
			Hint:            "Use skills_list to see all available skills",
		})
	}

	if strings.TrimSpace(in.FilePath) != "" {
		return json.Marshal(t.readLinkedFile(doc, in.FilePath))
	}
	return json.Marshal(t.readSkill(ctx, doc))
}

func (t *SkillViewTool) readSkill(ctx context.Context, doc skillToolDoc) skillViewToolResult {
	path := doc.Skill.Path
	raw, err := os.ReadFile(path)
	if err != nil {
		return skillViewError("failed to read skill: " + err.Error())
	}
	content := string(raw)
	skillDir := filepath.Dir(path)
	rendered, err := skills.PreprocessSkillContent(ctx, content, skills.PreprocessOptions{SkillDir: skillDir})
	if err == nil {
		content = rendered
	}
	linked := collectSkillLinkedFiles(skillDir)
	result := skillViewToolResult{
		Success:         true,
		Name:            doc.Skill.Name,
		Description:     doc.Skill.Description,
		Tags:            append([]string(nil), doc.Skill.HermesTags...),
		RelatedSkills:   append([]string(nil), doc.Skill.RelatedSkills...),
		Content:         content,
		Path:            doc.Skill.Path,
		SkillDir:        skillDir,
		LinkedFiles:     linked,
		ReadinessStatus: "available",
	}
	if len(linked) > 0 {
		result.UsageHint = "To view linked files, call skill_view(name, file_path) where file_path is e.g. 'references/api.md' or 'assets/config.yaml'"
	}
	return result
}

func (t *SkillViewTool) readLinkedFile(doc skillToolDoc, filePath string) skillViewToolResult {
	skillDir := filepath.Dir(doc.Skill.Path)
	rel, ok := cleanSkillRelativePath(filePath)
	if !ok {
		return skillViewToolResult{
			Success: false,
			Name:    doc.Skill.Name,
			Error:   "Path traversal ('..') is not allowed.",
			Hint:    "Use a relative path within the skill directory",
		}
	}
	target := filepath.Join(skillDir, filepath.FromSlash(rel))
	if !pathWithinDir(skillDir, target) {
		return skillViewToolResult{
			Success: false,
			Name:    doc.Skill.Name,
			Error:   "requested file is outside the skill directory",
			Hint:    "Use a relative path within the skill directory",
		}
	}
	raw, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return skillViewToolResult{
			Success:        false,
			Name:           doc.Skill.Name,
			Error:          fmt.Sprintf("File %q not found in skill %q.", rel, doc.Skill.Name),
			AvailableFiles: collectSkillLinkedFiles(skillDir),
			Hint:           "Use one of the available file paths listed above",
		}
	}
	if err != nil {
		return skillViewError("failed to read linked skill file: " + err.Error())
	}
	if !utf8.Valid(raw) {
		return skillViewToolResult{
			Success:  true,
			Name:     doc.Skill.Name,
			File:     rel,
			Content:  fmt.Sprintf("[Binary file: %s, size: %d bytes]", filepath.Base(target), len(raw)),
			IsBinary: true,
		}
	}
	return skillViewToolResult{
		Success:  true,
		Name:     doc.Skill.Name,
		File:     rel,
		Content:  string(raw),
		FileType: filepath.Ext(target),
	}
}

func normalizeSkillsToolsConfig(cfg SkillsToolsConfig) SkillsToolsConfig {
	if cfg.Root == "" {
		cfg.Root = skills.DefaultRoot()
	}
	if cfg.MaxDocumentBytes <= 0 {
		cfg.MaxDocumentBytes = skills.DefaultMaxDocumentBytes
	}
	return cfg
}

func loadSkillToolDocs(cfg SkillsToolsConfig) ([]skillToolDoc, error) {
	cfg = normalizeSkillsToolsConfig(cfg)
	var docs []skillToolDoc
	if cfg.Root != "" {
		store := skills.NewStore(cfg.Root, cfg.MaxDocumentBytes)
		snapshot, err := store.SnapshotActive()
		if err != nil {
			return nil, err
		}
		for _, skill := range snapshot.Skills {
			meta := readSkillToolMeta(skill.Path)
			category := firstSkillToolNonBlank(meta.Category, categoryFromSkillToolPath(store.ActiveDir(), skill.Path))
			source := firstSkillToolNonBlank(meta.Source, "local")
			trust := firstSkillToolNonBlank(meta.Trust, defaultSkillToolTrust(source))
			docs = append(docs, skillToolDoc{
				Skill:    skill,
				Category: category,
				Source:   source,
				Trust:    trust,
				Status:   string(skills.SkillStatusEnabled),
			})
		}
	}
	if strings.TrimSpace(cfg.BundledRoot) != "" {
		bundled, err := skills.LoadSkillDocs(cfg.BundledRoot, cfg.MaxDocumentBytes)
		if err != nil {
			return nil, err
		}
		for _, skill := range bundled {
			meta := readSkillToolMeta(skill.Path)
			category := firstSkillToolNonBlank(meta.Category, categoryFromSkillToolPath(cfg.BundledRoot, skill.Path))
			source := firstSkillToolNonBlank(meta.Source, "builtin")
			trust := firstSkillToolNonBlank(meta.Trust, "system")
			docs = append(docs, skillToolDoc{
				Skill:    skill,
				Category: category,
				Source:   source,
				Trust:    trust,
				Status:   string(skills.SkillStatusEnabled),
			})
		}
	}
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].Category != docs[j].Category {
			return docs[i].Category < docs[j].Category
		}
		return docs[i].Skill.Name < docs[j].Skill.Name
	})
	return docs, nil
}

func readSkillToolMeta(skillPath string) skillToolMeta {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(skillPath), "meta.json"))
	if err != nil {
		return skillToolMeta{}
	}
	var meta skillToolMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return skillToolMeta{}
	}
	return meta
}

func categoryFromSkillToolPath(root, skillPath string) string {
	if root == "" || skillPath == "" {
		return ""
	}
	rel, err := filepath.Rel(root, filepath.Dir(skillPath))
	if err != nil || rel == "." {
		return ""
	}
	parts := strings.FieldsFunc(filepath.ToSlash(rel), func(r rune) bool { return r == '/' })
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

func defaultSkillToolTrust(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "builtin":
		return "system"
	case "hub", "community":
		return "community"
	default:
		return "local"
	}
}

func firstSkillToolNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sortedSkillToolCategories(categories map[string]struct{}) []string {
	out := make([]string, 0, len(categories))
	for category := range categories {
		out = append(out, category)
	}
	sort.Strings(out)
	return out
}

func findSkillToolDoc(docs []skillToolDoc, name string) (skillToolDoc, bool) {
	want := normalizeSkillToolName(name)
	for _, doc := range docs {
		if normalizeSkillToolName(doc.Skill.Name) == want {
			return doc, true
		}
		if doc.Category != "" && normalizeSkillToolName(doc.Category+"/"+doc.Skill.Name) == want {
			return doc, true
		}
	}
	return skillToolDoc{}, false
}

func normalizeSkillToolName(name string) string {
	name = strings.TrimSpace(filepath.ToSlash(name))
	name = strings.TrimSuffix(name, "/SKILL.md")
	name = strings.TrimSuffix(name, "/skill.md")
	name = strings.TrimSuffix(name, ".md")
	return strings.ToLower(name)
}

func firstSkillToolNames(docs []skillToolDoc, limit int) []string {
	if limit <= 0 || len(docs) == 0 {
		return nil
	}
	out := make([]string, 0, min(limit, len(docs)))
	for i, doc := range docs {
		if i >= limit {
			break
		}
		out = append(out, doc.Skill.Name)
	}
	return out
}

func collectSkillLinkedFiles(skillDir string) map[string][]string {
	groups := map[string][]string{
		"references": {},
		"templates":  {},
		"assets":     {},
		"scripts":    {},
		"other":      {},
	}
	_ = filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "SKILL.md" || filepath.Base(path) == "meta.json" {
			return nil
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		switch {
		case strings.HasPrefix(rel, "references/"):
			groups["references"] = append(groups["references"], rel)
		case strings.HasPrefix(rel, "templates/"):
			groups["templates"] = append(groups["templates"], rel)
		case strings.HasPrefix(rel, "assets/"):
			groups["assets"] = append(groups["assets"], rel)
		case strings.HasPrefix(rel, "scripts/"):
			groups["scripts"] = append(groups["scripts"], rel)
		default:
			groups["other"] = append(groups["other"], rel)
		}
		return nil
	})
	out := make(map[string][]string)
	for key, values := range groups {
		if len(values) == 0 {
			continue
		}
		sort.Strings(values)
		out[key] = values
	}
	return out
}

func cleanSkillRelativePath(path string) (string, bool) {
	normalized := strings.TrimSpace(filepath.ToSlash(path))
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", false
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return "", false
		}
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(normalized)))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	return cleaned, true
}

func pathWithinDir(dir, target string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	if realDir, err := filepath.EvalSymlinks(absDir); err == nil {
		absDir = realDir
	}
	if realTarget, err := filepath.EvalSymlinks(absTarget); err == nil {
		absTarget = realTarget
	}
	rel, err := filepath.Rel(absDir, absTarget)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func skillViewError(message string) skillViewToolResult {
	return skillViewToolResult{Success: false, Error: message}
}
