package skills

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Snapshot struct {
	Skills []Skill
}

type Store struct {
	root     string
	maxBytes int
}

type Runtime struct {
	store        *Store
	selectionCap int
	usage        *UsageLogger
	externalDirs []string
	snapshot     *promptSnapshotWriter
}

type RuntimeConfig struct {
	Root               string
	ExternalDirs       []string
	MaxDocumentBytes   int
	SelectionCap       int
	UsageLogPath       string
	PromptSnapshotPath string
}

func NewStore(root string, maxBytes int) *Store {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDocumentBytes
	}
	return &Store{root: root, maxBytes: maxBytes}
}

func (s *Store) ActiveDir() string {
	if s == nil {
		return ""
	}
	return filepath.Join(s.root, "active")
}

func (s *Store) SnapshotActive() (Snapshot, error) {
	if s == nil {
		return Snapshot{}, nil
	}
	return snapshotSkillTree(s.ActiveDir(), s.maxBytes)
}

func NewRuntime(root string, maxBytes, selectionCap int, usageLogPath string) *Runtime {
	return NewRuntimeWithConfig(RuntimeConfig{
		Root:             root,
		MaxDocumentBytes: maxBytes,
		SelectionCap:     selectionCap,
		UsageLogPath:     usageLogPath,
	})
}

func NewRuntimeWithConfig(cfg RuntimeConfig) *Runtime {
	if cfg.SelectionCap <= 0 {
		cfg.SelectionCap = DefaultSelectionCap
	}
	return &Runtime{
		store:        NewStore(cfg.Root, cfg.MaxDocumentBytes),
		selectionCap: cfg.SelectionCap,
		usage:        NewUsageLogger(cfg.UsageLogPath),
		externalDirs: append([]string(nil), cfg.ExternalDirs...),
		snapshot:     newPromptSnapshotWriter(cfg.PromptSnapshotPath),
	}
}

func (r *Runtime) BuildSkillBlock(_ context.Context, userMessage string) (string, []string, error) {
	if r == nil || r.store == nil {
		return "", nil, nil
	}
	snapshot, err := r.SnapshotDiscovered()
	if err != nil {
		return "", nil, err
	}
	selected := Select(snapshot.Skills, userMessage, r.selectionCap)
	block := RenderBlock(selected)
	if err := r.writePromptSnapshot(userMessage, block, selected); err != nil {
		return "", nil, err
	}
	return block, skillNames(selected), nil
}

func (r *Runtime) RecordSkillUsage(ctx context.Context, skillNames []string) error {
	if r == nil || r.usage == nil {
		return nil
	}
	return r.usage.Record(ctx, skillNames)
}

func (r *Runtime) SnapshotDiscovered() (Snapshot, error) {
	if r == nil || r.store == nil {
		return Snapshot{}, nil
	}

	local, err := r.store.SnapshotActive()
	if err != nil {
		return Snapshot{}, err
	}

	out := Snapshot{Skills: append([]Skill(nil), local.Skills...)}
	seen := make(map[string]struct{}, len(out.Skills))
	for _, skill := range out.Skills {
		key := normalizedSkillName(skill.Name)
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}

	for _, root := range resolveExternalDirs(r.externalDirs) {
		snapshot, err := snapshotSkillTree(root, r.store.maxBytes)
		if err != nil {
			return Snapshot{}, err
		}
		for _, skill := range snapshot.Skills {
			key := normalizedSkillName(skill.Name)
			if key == "" {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out.Skills = append(out.Skills, skill)
		}
	}

	sort.Slice(out.Skills, func(i, j int) bool {
		if out.Skills[i].Name != out.Skills[j].Name {
			return out.Skills[i].Name < out.Skills[j].Name
		}
		return out.Skills[i].Path < out.Skills[j].Path
	})
	return out, nil
}

func snapshotSkillTree(root string, maxBytes int) (Snapshot, error) {
	info, err := os.Stat(root)
	switch {
	case os.IsNotExist(err):
		return Snapshot{}, nil
	case err != nil:
		return Snapshot{}, err
	case !info.IsDir():
		return Snapshot{}, fmt.Errorf("skills: path %q is not a directory", root)
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
		return Snapshot{}, err
	}
	sort.Strings(paths)

	out := Snapshot{Skills: make([]Skill, 0, len(paths))}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Snapshot{}, err
		}
		skill, err := Parse(raw, maxBytes)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%s: %w", path, err)
		}
		skill.Path = path
		out.Skills = append(out.Skills, skill)
	}
	return out, nil
}

func resolveExternalDirs(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	out := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		resolved := expandSkillPath(root)
		if resolved == "" {
			continue
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	return out
}

func expandSkillPath(root string) string {
	root = strings.TrimSpace(os.ExpandEnv(root))
	if root == "" {
		return ""
	}
	if root == "~" || strings.HasPrefix(root, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			if root == "~" {
				root = home
			} else {
				root = filepath.Join(home, root[2:])
			}
		}
	}
	return filepath.Clean(root)
}

func normalizedSkillName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (r *Runtime) writePromptSnapshot(userMessage, block string, skills []Skill) error {
	if r == nil || r.snapshot == nil {
		return nil
	}
	return r.snapshot.Write(userMessage, block, skills)
}
