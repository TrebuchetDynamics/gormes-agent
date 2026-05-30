package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/access"

// TrustClass is the trust label attached to a caller (channel, system path,
// or child agent). The MCP host boundary uses these labels to decide whether
// a tool declaration may be exposed or invoked. The string values match the
// progress.json trust_class vocabulary used elsewhere in Gormes (operator,
// gateway, child-agent, system).
type TrustClass = access.TrustClass

// Trust class constants. Keep these in sync with the progress.json
// trust_class enum and with subagent.TrustClass.
const (
	TrustClassOperator   = access.TrustClassOperator
	TrustClassGateway    = access.TrustClassGateway
	TrustClassChildAgent = access.TrustClassChildAgent
	TrustClassSystem     = access.TrustClassSystem
)

type TrustClassTool = access.TrustClassTool

type TrustClassExecutor = access.TrustClassExecutor

func NewTrustClassExecutor() *TrustClassExecutor {
	return access.NewTrustClassExecutor()
}
