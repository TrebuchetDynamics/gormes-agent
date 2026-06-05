package runtime

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/availability"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/candidate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/commands"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/selection"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/usage"
)

type Snapshot struct {
	Skills  []document.Skill
	Invalid []InvalidSkill
}

// InvalidSkill records one SKILL.md file the loader could not promote
// because its frontmatter failed structural validation. The file is omitted
// from prompt injection; callers surface the evidence as a SkillStatus.
type InvalidSkill struct {
	Path   string
	Errors []document.SkillValidationError
}

type Store struct {
	root     string
	maxBytes int
}

type Runtime struct {
	store        *Store
	selectionCap int
	usage        *usage.UsageLogger
}

func NewStore(root string, maxBytes int) *Store {
	if maxBytes <= 0 {
		maxBytes = document.DefaultMaxDocumentBytes
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

	activeDir := s.ActiveDir()
	info, err := os.Stat(activeDir)
	switch {
	case os.IsNotExist(err):
		return Snapshot{}, nil
	case err != nil:
		return Snapshot{}, err
	case !info.IsDir():
		return Snapshot{}, fmt.Errorf("skills: active path %q is not a directory", activeDir)
	}

	var paths []string
	if err := filepath.WalkDir(activeDir, func(path string, d fs.DirEntry, err error) error {
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

	out := Snapshot{Skills: make([]document.Skill, 0, len(paths))}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Snapshot{}, err
		}
		expectedSlug := filepath.Base(filepath.Dir(path))
		if errs := document.ValidateSkillFrontmatter(raw, document.FrontmatterValidateOptions{ExpectedSlug: expectedSlug}); len(errs) > 0 {
			out.Invalid = append(out.Invalid, InvalidSkill{Path: path, Errors: errs})
			continue
		}
		skill, err := document.Parse(raw, s.maxBytes)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%s: %w", path, err)
		}
		skill.Path = path
		out.Skills = append(out.Skills, skill)
	}
	return out, nil
}

func (s *Store) CandidateDir() string {
	if s == nil {
		return ""
	}
	return candidate.CandidateDir(s.root)
}

func (s *Store) DraftCandidate(draft candidate.CandidateDraft) (candidate.CandidateMetadata, error) {
	if s == nil {
		return candidate.CandidateMetadata{}, fmt.Errorf("skills: nil store")
	}
	return candidate.DraftCandidate(s.root, s.maxBytes, draft)
}

func (s *Store) PromoteCandidate(candidateID string) (candidate.ActiveMetadata, error) {
	if s == nil {
		return candidate.ActiveMetadata{}, fmt.Errorf("skills: nil store")
	}
	return candidate.PromoteCandidate(s.root, s.ActiveDir(), s.maxBytes, candidateID)
}

func NewRuntime(root string, maxBytes, selectionCap int, usageLogPath string) *Runtime {
	if selectionCap <= 0 {
		selectionCap = document.DefaultSelectionCap
	}
	return &Runtime{
		store:        NewStore(root, maxBytes),
		selectionCap: selectionCap,
		usage:        usage.NewUsageLogger(usageLogPath),
	}
}

func (r *Runtime) BuildSkillBlock(ctx context.Context, userMessage string) (string, []string, error) {
	block, names, _, err := r.BuildSkillBlockWithOptions(ctx, userMessage, availability.RuntimeOptions{})
	return block, names, err
}

func (r *Runtime) BuildSkillBlockWithOptions(ctx context.Context, userMessage string, opts availability.RuntimeOptions) (string, []string, []availability.SkillStatus, error) {
	if r == nil || r.store == nil {
		return "", nil, nil, nil
	}
	snapshot, err := r.store.SnapshotActive()
	if err != nil {
		return "", nil, nil, err
	}
	prepared, statuses := availability.PrepareSkills(ctx, snapshot.Skills, opts)
	for _, invalid := range snapshot.Invalid {
		statuses = append(statuses, invalidSkillStatus(invalid))
	}
	selected := selection.Select(prepared, userMessage, r.selectionCap)
	return document.RenderBlock(selected), skillNames(selected), statuses, nil
}

func invalidSkillStatus(invalid InvalidSkill) availability.SkillStatus {
	name := filepath.Base(filepath.Dir(invalid.Path))
	codes := make([]string, 0, len(invalid.Errors))
	for _, err := range invalid.Errors {
		codes = append(codes, string(err.Code))
	}
	return availability.SkillStatus{
		Name:   name,
		Path:   invalid.Path,
		Status: availability.SkillStatusFrontmatterInvalid,
		Reason: "frontmatter validation failed: " + strings.Join(codes, ", "),
	}
}

func (r *Runtime) RecordSkillUsage(ctx context.Context, skillNames []string) error {
	if r == nil || r.usage == nil {
		return nil
	}
	return r.usage.Record(ctx, skillNames)
}

func (r *Runtime) SkillSlashCommands(ctx context.Context, opts availability.RuntimeOptions) ([]commands.SkillSlashCommand, []availability.SkillStatus, error) {
	if r == nil || r.store == nil {
		return nil, nil, nil
	}
	snapshot, err := r.store.SnapshotActive()
	if err != nil {
		return nil, nil, err
	}
	prepared, statuses := availability.PrepareSkills(ctx, snapshot.Skills, opts)
	return commands.BuildSkillSlashCommands(prepared), statuses, nil
}

func skillNames(skills []document.Skill) []string {
	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		out = append(out, skill.Name)
	}
	return out
}
