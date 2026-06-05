package modelpolicy

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/text"
)

// PromptRole is the API-boundary role requested for system-prompt content.
// Internal Gormes transcript messages keep role=system; provider adapters may
// translate to developer when this helper requests it.
type PromptRole string

const (
	PromptRoleSystem    PromptRole = "system"
	PromptRoleDeveloper PromptRole = "developer"
)

// ModelPromptRole returns the provider-boundary role for a model name using
// Hermes' DEVELOPER_ROLE_MODELS substring policy.
func ModelPromptRole(model string) PromptRole {
	if matchModelFamily(model, text.DeveloperRoleModels) {
		return PromptRoleDeveloper
	}
	return PromptRoleSystem
}

// ModelPromptGuidanceOptions is a pure input bag for model-family prompt
// guidance. ToolUseEnforcementMode intentionally accepts bool/string/[]string
// so callers can pass decoded config values without binding this helper to the
// config package.
type ModelPromptGuidanceOptions struct {
	Model                  string
	ValidToolNames         []string
	ToolUseEnforcementMode any
}

// ModelPromptGuidanceResult describes the pure prompt blocks and role metadata
// selected for the model. Evidence strings are redacted status codes suitable
// for higher-level gateway/provider diagnostics.
type ModelPromptGuidanceResult struct {
	PromptRole                    PromptRole
	Guidance                      string
	Evidence                      []string
	ToolUseEnforcementDefaulted   bool
	ToolUseEnforcementShouldApply bool
}

// BuildModelPromptGuidance selects Hermes-compatible model guidance blocks
// without reading env/config, looking up live tools, or calling providers.
func BuildModelPromptGuidance(opts ModelPromptGuidanceOptions) ModelPromptGuidanceResult {
	result := ModelPromptGuidanceResult{PromptRole: ModelPromptRole(opts.Model)}
	var blocks []string

	applyToolUse, defaulted := shouldApplyToolUseEnforcement(opts.Model, opts.ToolUseEnforcementMode)
	result.ToolUseEnforcementDefaulted = defaulted
	result.ToolUseEnforcementShouldApply = applyToolUse
	if defaulted {
		result.Evidence = append(result.Evidence, "tool_use_enforcement_defaulted")
	}
	if applyToolUse {
		if len(opts.ValidToolNames) == 0 {
			result.Evidence = append(result.Evidence, "tool_use_enforcement_suppressed_no_tools")
		} else {
			blocks = append(blocks, text.ToolUseEnforcementGuidance)
		}
	}

	if matchModelFamily(opts.Model, []string{"gpt", "codex"}) {
		blocks = append(blocks, text.OpenAIModelExecutionGuidance)
	}
	if matchModelFamily(opts.Model, []string{"gemini", "gemma"}) {
		blocks = append(blocks, text.GoogleModelOperationalGuidance)
	}
	if hasValidTool(opts.ValidToolNames, "web_search") {
		blocks = append(blocks, text.ResearchQualityGuidance)
		result.Evidence = append(result.Evidence, "research_quality_guidance_injected")
	} else if hasAnyValidTool(opts.ValidToolNames, "web_extract", "web_crawl") {
		result.Evidence = append(result.Evidence, "research_quality_guidance_suppressed_no_web_search")
	}

	result.Guidance = strings.Join(blocks, "\n\n")
	return result
}

func shouldApplyToolUseEnforcement(model string, mode any) (bool, bool) {
	switch v := mode.(type) {
	case nil:
		return matchModelFamily(model, text.ToolUseEnforcementModels), false
	case bool:
		return v, false
	case string:
		m := strings.TrimSpace(strings.ToLower(v))
		switch m {
		case "", "auto":
			return matchModelFamily(model, text.ToolUseEnforcementModels), false
		case "true", "always", "on", "yes":
			return true, false
		case "false", "never", "off", "no":
			return false, false
		default:
			return matchModelFamily(model, []string{m}), false
		}
	case []string:
		return matchModelFamily(model, v), false
	case []any:
		families := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return matchModelFamily(model, text.ToolUseEnforcementModels), true
			}
			families = append(families, s)
		}
		return matchModelFamily(model, families), false
	default:
		return matchModelFamily(model, text.ToolUseEnforcementModels), true
	}
}

func matchModelFamily(model string, families []string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	for _, family := range families {
		needle := strings.ToLower(strings.TrimSpace(family))
		if needle != "" && strings.Contains(m, needle) {
			return true
		}
	}
	return false
}

func hasValidTool(tools []string, name string) bool {
	for _, tool := range tools {
		if strings.EqualFold(strings.TrimSpace(tool), name) {
			return true
		}
	}
	return false
}

func hasAnyValidTool(tools []string, names ...string) bool {
	for _, name := range names {
		if hasValidTool(tools, name) {
			return true
		}
	}
	return false
}
