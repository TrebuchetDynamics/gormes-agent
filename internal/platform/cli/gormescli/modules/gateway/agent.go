package gateway

import (
	"github.com/spf13/cobra"

	agentcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/agent"
)

type AgentCommandOptions = agentcmd.AgentCommandOptions
type AgentResetOptions = agentcmd.AgentResetOptions
type AgentSpawnOptions = agentcmd.AgentSpawnOptions
type AgentBindingMatch = agentcmd.AgentBindingMatch
type AgentCommandSeams = agentcmd.AgentCommandSeams

func NewAgentCommandWithSeams(seams AgentCommandSeams, opts AgentCommandOptions) *cobra.Command {
	return agentcmd.NewAgentCommandWithSeams(seams, opts)
}
