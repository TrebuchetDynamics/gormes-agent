package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/catalog"

const (
	SkillStatusEnabled SkillStatusCode = catalog.SkillStatusEnabled
)

type SkillRow = catalog.SkillRow

type ListOptions = catalog.ListOptions

func ListInstalledSkills(opts ListOptions, disabled map[string]struct{}) []SkillRow {
	return catalog.ListInstalledSkills(opts, disabled)
}

func ListInstalledSkillsFromRoots(root, bundledRoot string, opts ListOptions, disabled map[string]struct{}) []SkillRow {
	return catalog.ListInstalledSkillsFromRoots(root, bundledRoot, opts, disabled)
}

func BundledRoot() string { return catalog.BundledRoot() }

func LoadSkillDocs(root string, maxBytes int) ([]Skill, error) {
	return catalog.LoadSkillDocs(root, maxBytes)
}

func DefaultRoot() string { return catalog.DefaultRoot() }

func defaultSkillsRoot() string { return catalog.DefaultRoot() }
func bundledSkillsRoot() string { return catalog.BundledRoot() }
