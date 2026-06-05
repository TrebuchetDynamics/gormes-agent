package contracts

import (
	"context"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
)

// AgentsRegistry is the dynamic-agent registry seam used by the admin Agents
// screen. It matches Goncho's registry shape without coupling screen callers to
// the concrete memory-backed implementation.
type AgentsRegistry interface {
	List(context.Context) ([]goncho.AgentRecord, error)
	Create(context.Context, goncho.CreateAgentOptions) (goncho.AgentRecord, error)
	Bind(context.Context, string, goncho.BindingMatch) error
	Unbind(context.Context, goncho.BindingMatch) error
	Resolve(context.Context, goncho.BindingMatch) (string, bool, error)
}
