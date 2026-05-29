package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	gatewayrouting "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/routing"
)

type AgentBindingTier = gatewayrouting.AgentBindingTier

const (
	AgentBindingTierPeer       AgentBindingTier = gatewayrouting.AgentBindingTierPeer
	AgentBindingTierParentPeer AgentBindingTier = gatewayrouting.AgentBindingTierParentPeer
	AgentBindingTierGuildRoles AgentBindingTier = gatewayrouting.AgentBindingTierGuildRoles
	AgentBindingTierGuild      AgentBindingTier = gatewayrouting.AgentBindingTierGuild
	AgentBindingTierTeam       AgentBindingTier = gatewayrouting.AgentBindingTierTeam
	AgentBindingTierAccount    AgentBindingTier = gatewayrouting.AgentBindingTierAccount
	AgentBindingTierChannel    AgentBindingTier = gatewayrouting.AgentBindingTierChannel
	AgentBindingTierDefault    AgentBindingTier = gatewayrouting.AgentBindingTierDefault
)

type AgentRouteRequest = gatewayrouting.AgentRouteRequest

type AgentRouteDecision = gatewayrouting.AgentRouteDecision

type AgentRouter = gatewayrouting.AgentRouter

func NewAgentRouter(agents config.AgentsCfg, bindings []config.AgentBindingCfg) AgentRouter {
	return gatewayrouting.NewAgentRouter(agents, bindings)
}
