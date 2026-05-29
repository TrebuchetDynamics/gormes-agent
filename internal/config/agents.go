package config

import (
	"regexp"
	"strings"

	agentsconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/agents"
)

const DefaultAgentID = agentsconfig.DefaultAgentID

var agentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type AgentsCfg = agentsconfig.AgentsCfg
type AgentDefaultsCfg = agentsconfig.AgentDefaultsCfg
type AgentCfg = agentsconfig.AgentCfg
type AgentSandboxCfg = agentsconfig.AgentSandboxCfg
type AgentSandboxDockerCfg = agentsconfig.AgentSandboxDockerCfg
type AgentToolPolicy = agentsconfig.AgentToolPolicy
type AgentGroupChatCfg = agentsconfig.AgentGroupChatCfg
type AgentBindingCfg = agentsconfig.AgentBindingCfg
type AgentBindingMatchCfg = agentsconfig.AgentBindingMatchCfg
type AgentPeerMatchCfg = agentsconfig.AgentPeerMatchCfg

func defaultAgentsCfg(home string) AgentsCfg {
	return agentsconfig.DefaultConfig(home)
}

func normalizeAgentsConfig(home string, agents *AgentsCfg, bindings []AgentBindingCfg) error {
	return agentsconfig.NormalizeConfig(home, agents, bindings)
}

func normalizeAgentBindings(bindings []AgentBindingCfg) []AgentBindingCfg {
	return agentsconfig.NormalizeBindings(bindings)
}

func cleanStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
