package routing

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/fastmode"

type RequestOverrides = fastmode.RequestOverrides

func ResolveFastModeRequestOverrides(model string) (RequestOverrides, bool) {
	return fastmode.ResolveFastModeRequestOverrides(model)
}

func ModelSupportsAnthropicFastMode(model string) bool {
	return fastmode.ModelSupportsAnthropicFastMode(model)
}
