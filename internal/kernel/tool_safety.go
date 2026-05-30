package kernel

import kerneltoolsafety "github.com/TrebuchetDynamics/gormes-agent/internal/kernel/toolsafety"

type TrustClass = kerneltoolsafety.TrustClass

const (
	TrustClassOperator   = kerneltoolsafety.TrustClassOperator
	TrustClassSystem     = kerneltoolsafety.TrustClassSystem
	TrustClassGateway    = kerneltoolsafety.TrustClassGateway
	TrustClassChildAgent = kerneltoolsafety.TrustClassChildAgent
)

type ToolSafetyPolicy = kerneltoolsafety.ToolSafetyPolicy
type ToolSafetyDecision = kerneltoolsafety.ToolSafetyDecision
type OneshotToolSafetyOptions = kerneltoolsafety.OneshotToolSafetyOptions
type OneshotToolSafetyPolicy = kerneltoolsafety.OneshotToolSafetyPolicy
type AgentToolSafetyOptions = kerneltoolsafety.AgentToolSafetyOptions
type AgentToolSafetyPolicy = kerneltoolsafety.AgentToolSafetyPolicy

func ComposeToolSafetyPolicies(policies ...ToolSafetyPolicy) ToolSafetyPolicy {
	return kerneltoolsafety.ComposeToolSafetyPolicies(policies...)
}

func NewAgentToolSafetyPolicy(opts AgentToolSafetyOptions) ToolSafetyPolicy {
	return kerneltoolsafety.NewAgentToolSafetyPolicy(opts)
}

func NewOneshotToolSafetyPolicy(opts OneshotToolSafetyOptions) (*OneshotToolSafetyPolicy, error) {
	return kerneltoolsafety.NewOneshotToolSafetyPolicy(opts)
}
