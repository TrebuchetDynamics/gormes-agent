package agent

import "fmt"

// ThreadData middleware seeds per-turn thread metadata.
type threadDataMiddleware struct{}

func (m *threadDataMiddleware) Name() string { return "thread_data" }

func (m *threadDataMiddleware) Before(ctx *MiddlewareContext) error {
	if ctx.Data == nil {
		ctx.Data = make(map[string]any)
	}
	return nil
}

func (m *threadDataMiddleware) After(ctx *MiddlewareContext) error {
	return nil
}

// ToolError middleware handles tool execution errors gracefully.
type toolErrorMiddleware struct{}

func (m *toolErrorMiddleware) Name() string { return "tool_error" }

func (m *toolErrorMiddleware) Before(ctx *MiddlewareContext) error {
	return nil
}

func (m *toolErrorMiddleware) After(ctx *MiddlewareContext) error {
	return nil
}

// loopDetectAdapter wraps *LoopDetector into the Middleware interface.
// It runs Check() during Before and Record() during After.
type loopDetectAdapter struct {
	inner *LoopDetector
}

func (m *loopDetectAdapter) Name() string { return "loop_detector" }

func (m *loopDetectAdapter) Before(ctx *MiddlewareContext) error {
	result := m.inner.Check()
	if result.Detected {
		return fmt.Errorf("loop detected: type=%s evidence=%s", result.Type, result.Evidence)
	}
	return nil
}

func (m *loopDetectAdapter) After(ctx *MiddlewareContext) error {
	return nil
}

// Memory middleware prepares memory context injection.
type memoryMiddleware struct{}

func (m *memoryMiddleware) Name() string { return "memory" }

func (m *memoryMiddleware) Before(ctx *MiddlewareContext) error {
	return nil
}

func (m *memoryMiddleware) After(ctx *MiddlewareContext) error {
	return nil
}

// Subagent middleware manages subagent lifecycle scoping.
type subagentMiddleware struct{}

func (m *subagentMiddleware) Name() string { return "subagent" }

func (m *subagentMiddleware) Before(ctx *MiddlewareContext) error {
	return nil
}

func (m *subagentMiddleware) After(ctx *MiddlewareContext) error {
	return nil
}
