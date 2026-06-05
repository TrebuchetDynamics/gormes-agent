package cli

import skillscmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/skillscmd"

type (
	SkillsURLInstallDeps = skillscmd.SkillsURLInstallDeps
	SkillsCommandDeps    = skillscmd.SkillsCommandDeps
)

var (
	NewSkillsCommand        = skillscmd.NewSkillsCommand
	NewSkillsInstallCommand = skillscmd.NewSkillsInstallCommand
	NewSkillsListCommand    = skillscmd.NewSkillsListCommand
)
