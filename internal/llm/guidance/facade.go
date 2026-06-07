package guidance

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/conditional"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/modelpolicy"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/selfhelp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/text"
)

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

// DefaultSoulMD is the Gormes-owned port of Hermes' DEFAULT_SOUL_MD from
// hermes_cli/default_soul.py. The only intentional divergence is the product
// identity: Gorm is the editable default persona, while gormes is the
// Go-native Hermes-compatible runtime that runs it.
const DefaultSoulMD = text.DefaultSoulMD

const DefaultAgentIdentity = DefaultSoulMD

// ResearchQualityGuidance is Gormes-owned prompt guidance for external
// discovery, comparison, and recommendation tasks where shallow repo lists are
// worse than a source-backed migration or adoption strategy.
const ResearchQualityGuidance = text.ResearchQualityGuidance

// GuidanceSwitchResult is a generic result type for conditional guidance
// injection. It carries the guidance text if injected, along with evidence
// strings for diagnostics. Both MemoryGuidanceResult and
// SessionSearchGuidanceResult are type aliases of this struct.
type GuidanceSwitchResult = conditional.GuidanceSwitchResult

// MemoryGuidanceResult is the result of BuildMemoryGuidance.
// Deprecated: use GuidanceSwitchResult. This type alias is preserved for
// backward compatibility.
type MemoryGuidanceResult = GuidanceSwitchResult

// SessionSearchGuidanceResult is the result of BuildSessionSearchGuidance.
// Deprecated: use GuidanceSwitchResult. This type alias is preserved for
// backward compatibility.
type SessionSearchGuidanceResult = GuidanceSwitchResult

// buildGuidanceSwitch constructs a GuidanceSwitchResult. When enabled is
// true, the provided guidance text is returned with injected=true and the
// injectedEvidence string. When enabled is false, an empty result with the
// suppressedEvidence string is returned. Guidance is empty when not injected
// so callers can omit empty blocks.
func buildGuidanceSwitch(guidance string, enabled bool, suppressedEvidence, injectedEvidence string) GuidanceSwitchResult {
	return conditional.BuildGuidanceSwitch(guidance, enabled, suppressedEvidence, injectedEvidence)
}

func BuildMemoryGuidance(hasMemoryTool bool) MemoryGuidanceResult {
	return buildGuidanceSwitch(MemoryGuidance, hasMemoryTool,
		"memory_guidance_suppressed: no memory tool available",
		"memory_guidance_injected",
	)
}

func BuildSessionSearchGuidance(enabled bool) SessionSearchGuidanceResult {
	return buildGuidanceSwitch(SessionSearchGuidance, enabled,
		"session_search_guidance_suppressed: session search disabled",
		"session_search_guidance_injected",
	)
}

// PromptRole is the API-boundary role requested for system-prompt content.
// Internal Gormes transcript messages keep role=system; provider adapters may
// translate to developer when this helper requests it.
type PromptRole = modelpolicy.PromptRole

const (
	PromptRoleSystem    = modelpolicy.PromptRoleSystem
	PromptRoleDeveloper = modelpolicy.PromptRoleDeveloper
)

// ModelPromptRole returns the provider-boundary role for a model name using
// Hermes' DEVELOPER_ROLE_MODELS substring policy.
func ModelPromptRole(model string) PromptRole {
	return modelpolicy.ModelPromptRole(model)
}

// ModelPromptGuidanceOptions is a pure input bag for model-family prompt
// guidance. ToolUseEnforcementMode intentionally accepts bool/string/[]string
// so callers can pass decoded config values without binding this helper to the
// config package.
type ModelPromptGuidanceOptions = modelpolicy.ModelPromptGuidanceOptions

// ModelPromptGuidanceResult describes the pure prompt blocks and role metadata
// selected for the model. Evidence strings are redacted status codes suitable
// for higher-level gateway/provider diagnostics.
type ModelPromptGuidanceResult = modelpolicy.ModelPromptGuidanceResult

// BuildModelPromptGuidance selects Hermes-compatible model guidance blocks
// without reading env/config, looking up live tools, or calling providers.
func BuildModelPromptGuidance(opts ModelPromptGuidanceOptions) ModelPromptGuidanceResult {
	return modelpolicy.BuildModelPromptGuidance(opts)
}

// GormesSelfHelpGuidanceForPrompt returns the deterministic self-help prompt
// block only for user prompts that ask about operating Gormes itself.
func GormesSelfHelpGuidanceForPrompt(userPrompt string) (string, bool) {
	return selfhelp.GuidanceForPrompt(userPrompt)
}
