package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"

// MockTool is a test double kept at the root tools facade for existing tests
// and downstream callers. The implementation lives in toolkit with the core
// tool contract it satisfies.
type MockTool = toolkit.MockTool
