package subagent

import "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/policy"

// ErrDurableRouteDenied is returned when an untrusted caller attempts to
// submit privileged deterministic work through the durable-job route.
var ErrDurableRouteDenied = policy.ErrDurableRouteDenied

// TrustClass identifies who is asking the orchestration policy to submit work.
type TrustClass = policy.TrustClass

const (
	TrustOperator   = policy.TrustOperator
	TrustChildAgent = policy.TrustChildAgent
	TrustSystem     = policy.TrustSystem
)

// WorkKind is the coarse class of work being routed.
type WorkKind = policy.WorkKind

const (
	WorkKindShellCommand = policy.WorkKindShellCommand
	WorkKindCronJob      = policy.WorkKindCronJob
	WorkKindLLMSubagent  = policy.WorkKindLLMSubagent
)

// OrchestrationRoute is the execution route chosen by the policy.
type OrchestrationRoute = policy.OrchestrationRoute

const (
	RouteDurableJob   = policy.RouteDurableJob
	RouteLiveSubagent = policy.RouteLiveSubagent
	RouteDenied       = policy.RouteDenied
)

// OrchestrationLane keeps deterministic restartable work distinct from live
// LLM judgment loops while still reporting both through one control surface.
type OrchestrationLane = policy.OrchestrationLane

const (
	LaneDeterministic = policy.LaneDeterministic
	LaneLLMSubagent   = policy.LaneLLMSubagent
)

// ExecutionAPI names the Gormes-native API the selected route should use.
type ExecutionAPI = policy.ExecutionAPI

const (
	ExecutionAPIDurableJob   = policy.ExecutionAPIDurableJob
	ExecutionAPIDelegateTask = policy.ExecutionAPIDelegateTask
)

// ControlPlane names the operator-visible orchestration surface shared by
// deterministic durable jobs and live subagent runs.
type ControlPlane = policy.ControlPlane

const (
	ControlPlaneUnifiedOrchestrator = policy.ControlPlaneUnifiedOrchestrator
)

// MinionRoutingRequest is a pure policy input. It intentionally carries no
// queue/executor handles; this slice only decides where work belongs.
type MinionRoutingRequest = policy.MinionRoutingRequest

// MinionRoutingDecision describes the selected orchestration lane.
type MinionRoutingDecision = policy.MinionRoutingDecision

// MinionRoutingPolicy holds the Gormes trust matrix for durable jobs and live
// subagents while preserving Gormes-native delegate_task execution APIs.
type MinionRoutingPolicy = policy.MinionRoutingPolicy

// DefaultMinionRoutingPolicy returns the built-in routing policy.
func DefaultMinionRoutingPolicy() MinionRoutingPolicy {
	return policy.DefaultMinionRoutingPolicy()
}
