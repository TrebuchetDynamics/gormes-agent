package tools

import toolskills "github.com/TrebuchetDynamics/gormes-agent/internal/tools/skills"

const (
	SkillManagerToolName             = toolskills.SkillManagerToolName
	SkillWriteOriginForeground       = toolskills.SkillWriteOriginForeground
	SkillWriteOriginBackgroundReview = toolskills.SkillWriteOriginBackgroundReview
)

type SkillManagerToolConfig = toolskills.SkillManagerToolConfig
type SkillManagerTool = toolskills.SkillManagerTool

// NewSkillManagerTool returns a skill management tool.
func NewSkillManagerTool(cfg SkillManagerToolConfig) Tool {
	return toolskills.NewSkillManagerTool(cfg)
}
