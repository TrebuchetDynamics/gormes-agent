package agent

// FeatureFlag controls whether a middleware feature is active.
type FeatureFlag int

const (
	FeatureDisabled FeatureFlag = iota
	FeatureEnabled
)

// RuntimeFeatures declares which agent middlewares are active.
// true = use the built-in default middleware for that feature.
// false = disable that middleware entirely.
// Custom instances are supplied via CustomMiddleware.
type RuntimeFeatures struct {
	ThreadData FeatureFlag
	ToolError  FeatureFlag
	LoopDetect FeatureFlag
	Memory     FeatureFlag
	Subagent   FeatureFlag
	PlanMode   bool

	// CustomMiddleware overrides built-in middlewares by feature name.
	// Key is the built-in name (e.g. "loop_detector", "thread_data").
	// When set, the custom instance replaces the built-in.
	CustomMiddleware map[string]Middleware
}

// AssembleFromFeatures builds an ordered MiddlewareChain from RuntimeFeatures.
// Built-in ordering:
//
//  0. ThreadData (feature-gated)
//  1. ToolError  (feature-gated)
//  2. LoopDetect (feature-gated)
//  3. Memory     (feature-gated)
//  4. Subagent   (feature-gated, requires plan mode or separate flag)
//
// Disabled features are excluded entirely. Custom overrides replace built-ins.
func AssembleFromFeatures(features RuntimeFeatures) *MiddlewareChain {
	chain := NewMiddlewareChain()

	type featureDef struct {
		flag  FeatureFlag
		built func() Middleware
		name  string
	}

	defs := []featureDef{
		{flag: features.ThreadData, built: newThreadDataMiddleware, name: "thread_data"},
		{flag: features.ToolError, built: newToolErrorMiddleware, name: "tool_error"},
		{flag: features.LoopDetect, built: newLoopDetectMiddleware, name: "loop_detector"},
		{flag: features.Memory, built: newMemoryMiddleware, name: "memory"},
		{flag: features.Subagent, built: newSubagentMiddleware, name: "subagent"},
	}

	for _, d := range defs {
		if d.flag == FeatureDisabled {
			continue
		}
		if custom, ok := features.CustomMiddleware[d.name]; ok && custom != nil {
			chain.Add(custom)
		} else {
			chain.Add(d.built())
		}
	}

	return chain
}

// Built-in middleware factories.

func newThreadDataMiddleware() Middleware {
	return &threadDataMiddleware{}
}

func newToolErrorMiddleware() Middleware {
	return &toolErrorMiddleware{}
}

func newLoopDetectMiddleware() Middleware {
	return &loopDetectAdapter{inner: NewLoopDetector()}
}

func newMemoryMiddleware() Middleware {
	return &memoryMiddleware{}
}

func newSubagentMiddleware() Middleware {
	return &subagentMiddleware{}
}
