package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/execution"

type SkillCodeRequest = execution.SkillCodeRequest

type SkillCodeResult = execution.SkillCodeResult

type SkillCodeExecutionRequest = execution.SkillCodeExecutionRequest

type SkillCodeExecutionResult = execution.SkillCodeExecutionResult

type SkillCodeSandbox = execution.SkillCodeSandbox

type SkillCodeExecutor = execution.SkillCodeExecutor

func NewSkillCodeExecutor(sandbox SkillCodeSandbox) *SkillCodeExecutor {
	return execution.NewSkillCodeExecutor(sandbox)
}
