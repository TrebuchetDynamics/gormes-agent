package guidance

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/modelpolicy"

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
