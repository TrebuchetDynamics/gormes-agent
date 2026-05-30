package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/execution"

// ValidationResult captures the outcome of a skill validation run.
type ValidationResult = execution.ValidationResult

// SkillValidator runs lightweight sandbox execution to verify skills at load time.
// Broken skills are rejected unless the operator sets ForceLoad, which marks
// the result as ForceLoaded=true while preserving the original error.
type SkillValidator = execution.SkillValidator

// NewSkillValidator returns a SkillValidator that uses the given sandbox-backed executor.
func NewSkillValidator(executor *SkillCodeExecutor) *SkillValidator {
	return execution.NewSkillValidator(executor)
}

// ForceLoadMsg formats a human-readable message explaining that a skill was
// force-loaded despite validation failure.
func ForceLoadMsg(skillName string, validationErr string) string {
	return execution.ForceLoadMsg(skillName, validationErr)
}
