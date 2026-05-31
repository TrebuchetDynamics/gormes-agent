package guidance

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/text"

// MemoryGuidance is the upstream MEMORY_GUIDANCE constant.
// Source: ./hermes-agent/agent/prompt_builder.py MEMORY_GUIDANCE
const MemoryGuidance = text.MemoryGuidance

// SessionSearchGuidance is the upstream SESSION_SEARCH_GUIDANCE constant.
// Source: ./hermes-agent/agent/prompt_builder.py SESSION_SEARCH_GUIDANCE
const SessionSearchGuidance = text.SessionSearchGuidance

// SkillsGuidance is the upstream SKILLS_GUIDANCE constant.
// Source: ./hermes-agent/agent/prompt_builder.py SKILLS_GUIDANCE
const SkillsGuidance = text.SkillsGuidance

// ToolUseEnforcementGuidance is the upstream TOOL_USE_ENFORCEMENT_GUIDANCE constant.
// Source: ./hermes-agent/agent/prompt_builder.py TOOL_USE_ENFORCEMENT_GUIDANCE
const ToolUseEnforcementGuidance = text.ToolUseEnforcementGuidance

// ToolUseEnforcementModels is the upstream TOOL_USE_ENFORCEMENT_MODELS tuple.
// Source: ./hermes-agent/agent/prompt_builder.py TOOL_USE_ENFORCEMENT_MODELS
var ToolUseEnforcementModels = text.ToolUseEnforcementModels

// DeveloperRoleModels is the upstream DEVELOPER_ROLE_MODELS tuple.
// Source: ./hermes-agent/agent/prompt_builder.py DEVELOPER_ROLE_MODELS
var DeveloperRoleModels = text.DeveloperRoleModels

// OpenAIModelExecutionGuidance is the upstream OPENAI_MODEL_EXECUTION_GUIDANCE constant.
// Source: ./hermes-agent/agent/prompt_builder.py OPENAI_MODEL_EXECUTION_GUIDANCE
const OpenAIModelExecutionGuidance = text.OpenAIModelExecutionGuidance

// GoogleModelOperationalGuidance is the upstream GOOGLE_MODEL_OPERATIONAL_GUIDANCE constant.
// Source: ./hermes-agent/agent/prompt_builder.py GOOGLE_MODEL_OPERATIONAL_GUIDANCE
const GoogleModelOperationalGuidance = text.GoogleModelOperationalGuidance

// WSLEnvironmentHint is the upstream WSL_ENVIRONMENT_HINT constant.
// Source: ./hermes-agent/agent/prompt_builder.py WSL_ENVIRONMENT_HINT
const WSLEnvironmentHint = text.WSLEnvironmentHint
