package skills

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SkillReadinessStatus string

const (
	SkillReadinessReady     SkillReadinessStatus = "ready"
	SkillReadinessAvailable SkillReadinessStatus = "available"
)

type SkillMeta struct {
	Source      string               `json:"source"`
	Ref         string               `json:"ref"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Category    string               `json:"category,omitempty"`
	Version     string               `json:"version,omitempty"`
	Path        string               `json:"path,omitempty"`
	Optional    bool                 `json:"optional,omitempty"`
	Readiness   SkillReadinessStatus `json:"readiness"`
}

type SkillBundle struct {
	Name      string               `json:"name"`
	Title     string               `json:"title"`
	Prefix    string               `json:"prefix,omitempty"`
	Root      string               `json:"root,omitempty"`
	Optional  bool                 `json:"optional,omitempty"`
	Readiness SkillReadinessStatus `json:"readiness,omitempty"`
}

type HubLockFile struct {
	Version     int           `json:"version"`
	GeneratedAt time.Time     `json:"generated_at"`
	Bundles     []SkillBundle `json:"bundles"`
	Skills      []SkillMeta   `json:"skills"`
}

type SkillSource interface {
	Bundle() SkillBundle
	ListSkills(ctx context.Context) ([]SkillMeta, error)
}

type FilesystemSourceConfig struct {
	Name             string
	Title            string
	Prefix           string
	Root             string
	Optional         bool
	Readiness        SkillReadinessStatus
	MaxDocumentBytes int
}

type FilesystemSource struct {
	cfg FilesystemSourceConfig
}

func NewFilesystemSource(cfg FilesystemSourceConfig) *FilesystemSource {
	if cfg.Readiness == "" {
		cfg.Readiness = SkillReadinessReady
	}
	if cfg.MaxDocumentBytes <= 0 {
		cfg.MaxDocumentBytes = DefaultMaxDocumentBytes
	}
	return &FilesystemSource{cfg: cfg}
}

func (s *FilesystemSource) Bundle() SkillBundle {
	if s == nil {
		return SkillBundle{}
	}
	return SkillBundle{
		Name:      s.cfg.Name,
		Title:     s.cfg.Title,
		Prefix:    s.cfg.Prefix,
		Root:      s.cfg.Root,
		Optional:  s.cfg.Optional,
		Readiness: s.cfg.Readiness,
	}
}

func (s *FilesystemSource) ListSkills(ctx context.Context) ([]SkillMeta, error) {
	if s == nil {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Stat(s.cfg.Root)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	case !info.IsDir():
		return nil, fmt.Errorf("skills: source root %q is not a directory", s.cfg.Root)
	}

	var paths []string
	if err := filepath.WalkDir(s.cfg.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
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

	out := make([]SkillMeta, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		skill, err := Parse(raw, s.cfg.MaxDocumentBytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		ref, category, err := skillRef(s.cfg.Root, s.cfg.Prefix, path)
		if err != nil {
			return nil, err
		}
		out = append(out, SkillMeta{
			Source:      s.cfg.Name,
			Ref:         ref,
			Name:        skill.Name,
			Description: skill.Description,
			Category:    category,
			Path:        path,
			Optional:    s.cfg.Optional,
			Readiness:   s.cfg.Readiness,
		})
	}

	sortSkillMeta(out)
	return out, nil
}

type StaticSource struct {
	bundle SkillBundle
	skills []SkillMeta
}

func NewStaticSource(bundle SkillBundle, skills []SkillMeta) *StaticSource {
	out := make([]SkillMeta, len(skills))
	copy(out, skills)
	sortSkillMeta(out)
	return &StaticSource{
		bundle: bundle,
		skills: out,
	}
}

func (s *StaticSource) Bundle() SkillBundle {
	if s == nil {
		return SkillBundle{}
	}
	return s.bundle
}

func (s *StaticSource) ListSkills(ctx context.Context) ([]SkillMeta, error) {
	if s == nil {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]SkillMeta, len(s.skills))
	copy(out, s.skills)
	return out, nil
}

type SourceRegistry struct {
	sources map[string]SkillSource
	order   []string
}

func NewSourceRegistry(sources ...SkillSource) (*SourceRegistry, error) {
	reg := &SourceRegistry{
		sources: make(map[string]SkillSource, len(sources)),
		order:   make([]string, 0, len(sources)),
	}
	for _, source := range sources {
		if source == nil {
			continue
		}
		bundle := source.Bundle()
		if strings.TrimSpace(bundle.Name) == "" {
			return nil, fmt.Errorf("skills: source bundle name is required")
		}
		if _, exists := reg.sources[bundle.Name]; exists {
			return nil, fmt.Errorf("skills: duplicate source bundle %q", bundle.Name)
		}
		reg.sources[bundle.Name] = source
		reg.order = append(reg.order, bundle.Name)
	}
	sort.Strings(reg.order)
	return reg, nil
}

func (r *SourceRegistry) Bundles() []SkillBundle {
	if r == nil {
		return nil
	}
	out := make([]SkillBundle, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.sources[name].Bundle())
	}
	sortSkillBundles(out)
	return out
}

func (r *SourceRegistry) ListSkills(ctx context.Context) ([]SkillMeta, error) {
	if r == nil {
		return nil, nil
	}
	var out []SkillMeta
	for _, name := range r.order {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		skills, err := r.sources[name].ListSkills(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, skills...)
	}
	sortSkillMeta(out)
	return out, nil
}

func (r *SourceRegistry) Lock(ctx context.Context, generatedAt time.Time) (HubLockFile, error) {
	skills, err := r.ListSkills(ctx)
	if err != nil {
		return HubLockFile{}, err
	}
	return HubLockFile{
		Version:     1,
		GeneratedAt: generatedAt.UTC(),
		Bundles:     r.Bundles(),
		Skills:      skills,
	}, nil
}

func BuiltinSkillRegistryBundles() []SkillBundle {
	out := []SkillBundle{
		{Name: "bundled", Title: "Bundled skills", Prefix: "bundled"},
		{Name: "official", Title: "Official optional skills", Prefix: "official", Optional: true},
		{Name: "claude-marketplace", Title: "Claude Marketplace"},
		{Name: "clawhub", Title: "ClawHub"},
		{Name: "github", Title: "GitHub"},
		{Name: "hermes-index", Title: "Hermes Index"},
		{Name: "lobehub", Title: "LobeHub"},
		{Name: "skills.sh", Title: "skills.sh"},
	}
	sortSkillBundles(out)
	return out
}

func skillRef(root, prefix, path string) (ref, category string, err error) {
	relDir, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return "", "", err
	}
	relDir = filepath.ToSlash(relDir)
	if relDir == "." || relDir == "" {
		return "", "", fmt.Errorf("skills: path %q must be nested under a category directory", path)
	}
	parts := strings.Split(relDir, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("skills: path %q must include category and skill directories", path)
	}
	category = strings.Join(parts[:len(parts)-1], "/")
	ref = relDir
	if prefix = strings.TrimSpace(prefix); prefix != "" {
		ref = prefix + "/" + relDir
	}
	return ref, category, nil
}

func sortSkillMeta(skills []SkillMeta) {
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Ref != skills[j].Ref {
			return skills[i].Ref < skills[j].Ref
		}
		if skills[i].Source != skills[j].Source {
			return skills[i].Source < skills[j].Source
		}
		if skills[i].Category != skills[j].Category {
			return skills[i].Category < skills[j].Category
		}
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Path < skills[j].Path
	})
}

func sortSkillBundles(bundles []SkillBundle) {
	priority := map[string]int{
		"bundled":  0,
		"official": 1,
	}
	sort.Slice(bundles, func(i, j int) bool {
		pi, iok := priority[bundles[i].Name]
		pj, jok := priority[bundles[j].Name]
		switch {
		case iok && jok && pi != pj:
			return pi < pj
		case iok != jok:
			return iok
		}
		if bundles[i].Name != bundles[j].Name {
			return bundles[i].Name < bundles[j].Name
		}
		return bundles[i].Title < bundles[j].Title
	})
}
