package tools

import toolskills "github.com/TrebuchetDynamics/gormes-agent/internal/tools/skills"

const (
	SkillsListToolName = toolskills.SkillsListToolName
	SkillViewToolName  = toolskills.SkillViewToolName
)

type SkillsToolsConfig = toolskills.SkillsToolsConfig
type SkillsListTool = toolskills.SkillsListTool
type SkillViewTool = toolskills.SkillViewTool

// NewSkillsTools returns the built-in Hermes-compatible read-only skills tools.
func NewSkillsTools(cfg SkillsToolsConfig) []Tool {
	return toolskills.NewSkillsTools(cfg)
}

func NewSkillsListTool(cfg SkillsToolsConfig) *SkillsListTool {
	return toolskills.NewSkillsListTool(cfg)
}

func NewSkillViewTool(cfg SkillsToolsConfig) *SkillViewTool {
	return toolskills.NewSkillViewTool(cfg)
}
