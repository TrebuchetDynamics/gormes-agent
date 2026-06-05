// Package hermes ports unbranded Hermes prompt-builder guidance constants
// from upstream `agent/prompt_builder.py` for use by live-turn prompt assembly.
package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"

const MemoryGuidance = guidance.MemoryGuidance
const SessionSearchGuidance = guidance.SessionSearchGuidance
const SkillsGuidance = guidance.SkillsGuidance
const ToolUseEnforcementGuidance = guidance.ToolUseEnforcementGuidance

var ToolUseEnforcementModels = guidance.ToolUseEnforcementModels
var DeveloperRoleModels = guidance.DeveloperRoleModels

const OpenAIModelExecutionGuidance = guidance.OpenAIModelExecutionGuidance
const GoogleModelOperationalGuidance = guidance.GoogleModelOperationalGuidance
const WSLEnvironmentHint = guidance.WSLEnvironmentHint
