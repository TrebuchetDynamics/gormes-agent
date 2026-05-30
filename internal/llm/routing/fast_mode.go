package routing

import "strings"

// RequestOverrides carries provider-specific request knobs resolved from model routing.
type RequestOverrides struct {
	ServiceTier              string
	Speed                    string
	OpenRouterMinCodingScore string
}

// ResolveFastModeRequestOverrides returns provider-specific request overrides for
// models with known fast/priority runtime lanes.
func ResolveFastModeRequestOverrides(model string) (RequestOverrides, bool) {
	if modelSupportsAnthropicFastMode(model) {
		return RequestOverrides{Speed: "fast"}, true
	}
	if modelSupportsOpenAIPriorityProcessing(model) {
		return RequestOverrides{ServiceTier: "priority"}, true
	}
	return RequestOverrides{}, false
}

func modelSupportsOpenAIPriorityProcessing(model string) bool {
	base := fastModeModelBase(model)
	if base == "" || strings.Contains(base, "codex") {
		return false
	}
	for _, prefix := range []string{"gpt-", "o1", "o3", "o4"} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

// ModelSupportsAnthropicFastMode reports whether a model accepts Anthropic's
// fast-mode request field.
func ModelSupportsAnthropicFastMode(model string) bool {
	base := fastModeModelBase(model)
	if !strings.HasPrefix(base, "claude-") {
		return false
	}
	return strings.Contains(base, "opus-4-6") || strings.Contains(base, "opus-4.6")
}

func modelSupportsAnthropicFastMode(model string) bool {
	return ModelSupportsAnthropicFastMode(model)
}

func fastModeModelBase(model string) string {
	raw := strings.ToLower(strings.TrimSpace(model))
	if raw == "" {
		return ""
	}
	if slash := strings.Index(raw, "/"); slash >= 0 {
		raw = raw[slash+1:]
	}
	if colon := strings.Index(raw, ":"); colon >= 0 {
		raw = raw[:colon]
	}
	return raw
}
