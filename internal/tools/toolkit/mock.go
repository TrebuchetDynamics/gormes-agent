package toolkit

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit/testkit"

// MockTool is a test double. Every field is independently configurable so
// tests can script happy paths, slow executions, panics, or ctx-cancel
// scenarios. Not used in production code.
type MockTool = testkit.MockTool

var _ Tool = (*MockTool)(nil)
