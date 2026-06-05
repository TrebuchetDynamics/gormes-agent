package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/availability"
)

type SkillStatusCode = availability.SkillStatusCode

type SkillStatus = availability.SkillStatus

type RuntimeOptions = availability.RuntimeOptions

const (
	SkillStatusAvailable           = availability.SkillStatusAvailable
	SkillStatusDisabled            = availability.SkillStatusDisabled
	SkillStatusUnsupported         = availability.SkillStatusUnsupported
	SkillStatusMissingPrerequisite = availability.SkillStatusMissingPrerequisite
	SkillStatusPreprocessingFailed = availability.SkillStatusPreprocessingFailed
	SkillStatusFrontmatterInvalid  = availability.SkillStatusFrontmatterInvalid
	SkillStatusConditionExcluded   = availability.SkillStatusConditionExcluded
	SkillStatusPolicyExcluded      = availability.SkillStatusPolicyExcluded
)

func prepareSkills(ctx context.Context, in []Skill, opts RuntimeOptions) ([]Skill, []SkillStatus) {
	return availability.PrepareSkills(ctx, in, opts)
}

func missingSkillCredentials(skill Skill, env map[string]string) []string {
	return availability.MissingSkillCredentials(skill, env)
}

func skillConditionsMatch(conditions SkillConditions, availableTools, availableToolsets []string) bool {
	return availability.SkillConditionsMatch(conditions, availableTools, availableToolsets)
}
