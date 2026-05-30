package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/skillsconfig"

// SkillsCfg configures the Phase 2.G0 static skills runtime.
type SkillsCfg = skillsconfig.Config

const (
	SkillsExternalDirResolved = skillsconfig.ExternalDirResolved
	SkillsExternalDirSkipped  = skillsconfig.ExternalDirSkipped
)

type SkillsExternalDirEvidence = skillsconfig.ExternalDirEvidence

// SkillsRoot returns the root directory of the static skills runtime.
// Explicit override wins; otherwise the Gormes home default is used.
func (c Config) SkillsRoot() string {
	return skillsconfig.Root(c.Skills.Root)
}

// ExternalSkillsDirs resolves Hermes-compatible skills.external_dirs entries.
// Paths expand ~ and environment variables, relative entries resolve against
// GormesHome rather than process cwd, and invalid/duplicate/local roots are
// skipped with typed evidence instead of failing provider startup.
func (c Config) ExternalSkillsDirs() ([]string, []SkillsExternalDirEvidence) {
	return skillsconfig.ExternalDirs(c.SkillsRoot(), c.Skills.ExternalDirs)
}

// SkillsUsageLogPath returns the append-only JSONL path for skill usage.
// Explicit override wins; otherwise it lives under the skills root.
func (c Config) SkillsUsageLogPath() string {
	return skillsconfig.UsageLogPath(c.SkillsRoot(), c.Skills.UsageLogPath)
}
