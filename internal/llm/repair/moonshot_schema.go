package repair

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/moonshot"
)

// IsMoonshotModel reports whether a model slug targets Kimi/Moonshot, including
// aggregator-prefixed slugs such as openrouter/moonshotai/kimi-k2.
func IsMoonshotModel(model string) bool {
	return moonshot.IsMoonshotModel(model)
}

// SanitizeToolSchemaForModel applies provider-specific tool-parameter schema
// repairs when a model family needs stricter request shaping.
func SanitizeToolSchemaForModel(model string, raw json.RawMessage) json.RawMessage {
	return moonshot.SanitizeToolSchemaForModel(model, raw)
}

// SanitizeToolDescriptorsForModel returns a deep copy of descriptors using the
// provider-specific tool schema sanitizer selected by model.
func SanitizeToolDescriptorsForModel(model string, descriptors []ToolDescriptor) []ToolDescriptor {
	return moonshot.SanitizeToolDescriptorsForModel(model, descriptors)
}

// SanitizeMoonshotToolDescriptors returns provider-safe descriptors for
// Moonshot/Kimi's stricter flavored JSON Schema validator.
func SanitizeMoonshotToolDescriptors(descriptors []ToolDescriptor) []ToolDescriptor {
	return moonshot.SanitizeMoonshotToolDescriptors(descriptors)
}

// SanitizeMoonshotToolParameters normalizes tool parameters to the subset of
// JSON Schema accepted by Moonshot/Kimi tool calling.
func SanitizeMoonshotToolParameters(raw json.RawMessage) json.RawMessage {
	return moonshot.SanitizeMoonshotToolParameters(raw)
}
