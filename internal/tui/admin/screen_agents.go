package admin

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/agents"
	admincontracts "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/contracts"
)

// AgentsRegistry is the dynamic-agent registry seam used by the admin Agents
// screen. Root aliases preserve the original admin package API while the
// focused contracts package lets registry providers avoid depending on the
// concrete screen implementation.
type AgentsRegistry = admincontracts.AgentsRegistry

// AgentsScreen renders dynamic-agent spawn, bind, inspect, and unbind flows.
type AgentsScreen = agents.Screen

// AgentsOption configures an AgentsScreen.
type AgentsOption = agents.Option

// WithAgentsRegistry replaces the default memory-backed dynamic-agent registry.
func WithAgentsRegistry(registry AgentsRegistry) AgentsOption {
	return agents.WithRegistry(registry)
}

// NewAgentsScreen returns the admin Agents tab.
func NewAgentsScreen(opts ...AgentsOption) *AgentsScreen {
	return agents.NewScreen(opts...)
}
